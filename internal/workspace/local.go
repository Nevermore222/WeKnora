package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/google/uuid"
)

const registryFileName = ".xelora-workspaces.json"

const (
	StatusAvailable    = "available"
	StatusMissing      = "missing"
	StatusAccessDenied = "access_denied"
)

var (
	ErrNotConfigured      = errors.New("workspace root is not configured")
	ErrNotFound           = errors.New("workspace not found")
	ErrAlreadyExists      = errors.New("workspace already exists")
	ErrInvalidName        = errors.New("invalid workspace name")
	ErrPathEscape         = errors.New("workspace path escapes configured root")
	ErrAccessDenied       = errors.New("workspace access denied")
	ErrFileTooLarge       = errors.New("workspace file is too large to preview")
	ErrUnsupportedPreview = errors.New("workspace file type is not supported for inline preview")
)

type Entry struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	RelativePath string `json:"relative_path"`
	RootPath     string `json:"root_path,omitempty"`
	Status       string `json:"status"`
	TenantID     uint64 `json:"-"`
	UserID       string `json:"-"`
}

type CreateInput struct {
	Name string `json:"name"`
}

type FileEntry struct {
	Name         string    `json:"name"`
	RelativePath string    `json:"relative_path"`
	Kind         string    `json:"kind"`
	Size         int64     `json:"size"`
	ModifiedAt   time.Time `json:"modified_at"`
	IsDir        bool      `json:"is_dir"`
}

type FilePreview struct {
	Name         string `json:"name"`
	RelativePath string `json:"relative_path"`
	ContentType  string `json:"content_type"`
	Content      string `json:"content"`
	Size         int64  `json:"size"`
}

type FileOpenResult struct {
	Name         string
	RelativePath string
	AbsolutePath string
	ContentType  string
	Size         int64
}

type diskEntry struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	RelativePath string `json:"relative_path"`
	TenantID     uint64 `json:"tenant_id"`
	UserID       string `json:"user_id"`
}

type registryFile struct {
	Workspaces []diskEntry `json:"workspaces"`
}

type LocalRegistry struct {
	root string
	mu   sync.Mutex
}

func NewLocalRegistry(root string) *LocalRegistry {
	return &LocalRegistry{root: filepath.Clean(strings.TrimSpace(root))}
}

func (r *LocalRegistry) Create(ctx context.Context, input CreateInput) (*Entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name, err := validateDirectoryName(input.Name)
	if err != nil {
		return nil, err
	}
	if err := r.ensureRoot(); err != nil {
		return nil, err
	}
	tenantID, userID, err := scopeFromContext(ctx)
	if err != nil {
		return nil, err
	}
	state, err := r.load()
	if err != nil {
		return nil, err
	}
	for _, existing := range state.Workspaces {
		if existing.TenantID == tenantID && existing.UserID == userID && strings.EqualFold(existing.Name, name) {
			return nil, ErrAlreadyExists
		}
	}

	target := filepath.Join(r.root, name)
	if !isWithinRoot(r.root, target) {
		return nil, ErrPathEscape
	}
	if _, statErr := os.Lstat(target); statErr == nil {
		return nil, ErrAlreadyExists
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect workspace directory: %w", statErr)
	}
	if err := os.Mkdir(target, 0o755); err != nil {
		return nil, fmt.Errorf("create workspace directory: %w", err)
	}

	stored := diskEntry{
		ID:           uuid.NewString(),
		Name:         name,
		RelativePath: name,
		TenantID:     tenantID,
		UserID:       userID,
	}
	state.Workspaces = append(state.Workspaces, stored)
	if err := r.save(state); err != nil {
		_ = os.Remove(target)
		return nil, err
	}
	return publicEntry(stored, target, StatusAvailable, true), nil
}

func (r *LocalRegistry) List(ctx context.Context) ([]*Entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.ensureRoot(); err != nil {
		return nil, err
	}
	tenantID, userID, err := scopeFromContext(ctx)
	if err != nil {
		return nil, err
	}
	state, err := r.load()
	if err != nil {
		return nil, err
	}
	entries := make([]*Entry, 0)
	for _, stored := range state.Workspaces {
		if stored.TenantID != tenantID || stored.UserID != userID {
			continue
		}
		rootPath, status := r.entryStatus(stored)
		entries = append(entries, publicEntry(stored, rootPath, status, false))
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, nil
}

func (r *LocalRegistry) Resolve(ctx context.Context, id string) (*Entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.ensureRoot(); err != nil {
		return nil, err
	}
	tenantID, userID, err := scopeFromContext(ctx)
	if err != nil {
		return nil, err
	}
	state, err := r.load()
	if err != nil {
		return nil, err
	}
	for _, stored := range state.Workspaces {
		if stored.ID != strings.TrimSpace(id) || stored.TenantID != tenantID || stored.UserID != userID {
			continue
		}
		rootPath, err := r.resolvePath(stored)
		if err != nil {
			return nil, err
		}
		if err := probeWritable(rootPath); err != nil {
			return nil, err
		}
		return publicEntry(stored, rootPath, StatusAvailable, true), nil
	}
	return nil, ErrNotFound
}

func (r *LocalRegistry) ListFiles(ctx context.Context, workspaceID string, relDir string) ([]*FileEntry, error) {
	entry, dir, err := r.resolveFilePath(ctx, workspaceID, relDir)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, ErrInvalidName
	}
	children, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := make([]*FileEntry, 0, len(children))
	for _, child := range children {
		if strings.HasPrefix(child.Name(), ".") {
			continue
		}
		childInfo, err := child.Info()
		if err != nil {
			continue
		}
		abs := filepath.Join(dir, child.Name())
		rel, err := filepath.Rel(entry.RootPath, abs)
		if err != nil {
			continue
		}
		files = append(files, &FileEntry{
			Name:         child.Name(),
			RelativePath: filepath.ToSlash(rel),
			Kind:         detectFileKind(child.Name(), child.IsDir()),
			Size:         childInfo.Size(),
			ModifiedAt:   childInfo.ModTime(),
			IsDir:        child.IsDir(),
		})
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
	return files, nil
}

func (r *LocalRegistry) OpenFile(ctx context.Context, workspaceID string, relPath string) (*FileOpenResult, error) {
	entry, abs, err := r.resolveFilePath(ctx, workspaceID, relPath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if info.IsDir() {
		return nil, ErrInvalidName
	}
	rel, err := filepath.Rel(entry.RootPath, abs)
	if err != nil {
		return nil, ErrPathEscape
	}
	return &FileOpenResult{
		Name:         filepath.Base(abs),
		RelativePath: filepath.ToSlash(rel),
		AbsolutePath: abs,
		ContentType:  detectContentType(abs),
		Size:         info.Size(),
	}, nil
}

func (r *LocalRegistry) PreviewFile(ctx context.Context, workspaceID string, relPath string, maxBytes int64) (*FilePreview, error) {
	opened, err := r.OpenFile(ctx, workspaceID, relPath)
	if err != nil {
		return nil, err
	}
	if maxBytes > 0 && opened.Size > maxBytes {
		return nil, ErrFileTooLarge
	}
	if !isTextPreviewType(opened.RelativePath, opened.ContentType) {
		return nil, ErrUnsupportedPreview
	}
	data, err := os.ReadFile(opened.AbsolutePath)
	if err != nil {
		return nil, err
	}
	return &FilePreview{
		Name:         opened.Name,
		RelativePath: opened.RelativePath,
		ContentType:  opened.ContentType,
		Content:      string(data),
		Size:         opened.Size,
	}, nil
}

func (r *LocalRegistry) resolveFilePath(ctx context.Context, workspaceID string, relPath string) (*Entry, string, error) {
	entry, err := r.Resolve(ctx, workspaceID)
	if err != nil {
		return nil, "", err
	}
	cleanRel := filepath.Clean(strings.TrimSpace(relPath))
	if cleanRel == "." || cleanRel == "" {
		return entry, entry.RootPath, nil
	}
	if filepath.IsAbs(cleanRel) || cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) {
		return nil, "", ErrPathEscape
	}
	abs := filepath.Join(entry.RootPath, cleanRel)
	if !isWithinRoot(entry.RootPath, abs) {
		return nil, "", ErrPathEscape
	}
	if _, err := os.Lstat(abs); err == nil {
		realRoot, rootErr := filepath.EvalSymlinks(entry.RootPath)
		realTarget, targetErr := filepath.EvalSymlinks(abs)
		if rootErr != nil || targetErr != nil || !isWithinRoot(realRoot, realTarget) {
			return nil, "", ErrPathEscape
		}
		abs = realTarget
	}
	return entry, abs, nil
}

func (r *LocalRegistry) ensureRoot() error {
	if strings.TrimSpace(r.root) == "" || r.root == "." {
		return ErrNotConfigured
	}
	if err := os.MkdirAll(r.root, 0o755); err != nil {
		return fmt.Errorf("prepare workspace root: %w", err)
	}
	return nil
}

func (r *LocalRegistry) entryStatus(stored diskEntry) (string, string) {
	rootPath, err := r.resolvePath(stored)
	if err == nil {
		if probeErr := probeWritable(rootPath); probeErr == nil {
			return rootPath, StatusAvailable
		}
		return "", StatusAccessDenied
	}
	if errors.Is(err, ErrNotFound) {
		return "", StatusMissing
	}
	return "", StatusAccessDenied
}

func (r *LocalRegistry) resolvePath(stored diskEntry) (string, error) {
	target := filepath.Join(r.root, stored.RelativePath)
	if !isWithinRoot(r.root, target) {
		return "", ErrPathEscape
	}
	if _, err := os.Lstat(target); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("inspect workspace: %w", err)
	}
	realRoot, err := filepath.EvalSymlinks(r.root)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}
	realTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", fmt.Errorf("resolve workspace path: %w", err)
	}
	if !isWithinRoot(realRoot, realTarget) {
		return "", ErrPathEscape
	}
	info, err := os.Stat(realTarget)
	if err != nil {
		return "", fmt.Errorf("inspect workspace: %w", err)
	}
	if !info.IsDir() {
		return "", ErrNotFound
	}
	return realTarget, nil
}

func (r *LocalRegistry) load() (*registryFile, error) {
	data, err := os.ReadFile(filepath.Join(r.root, registryFileName))
	if errors.Is(err, os.ErrNotExist) {
		return &registryFile{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read workspace registry: %w", err)
	}
	var state registryFile
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decode workspace registry: %w", err)
	}
	return &state, nil
}

func (r *LocalRegistry) save(state *registryFile) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode workspace registry: %w", err)
	}
	data = append(data, '\n')
	temp, err := os.CreateTemp(r.root, ".xelora-workspaces-*.tmp")
	if err != nil {
		return fmt.Errorf("create workspace registry temp file: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, filepath.Join(r.root, registryFileName)); err != nil {
		return fmt.Errorf("replace workspace registry: %w", err)
	}
	return nil
}

func validateDirectoryName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" || name == "." || name == ".." || len(name) > 128 {
		return "", ErrInvalidName
	}
	if filepath.IsAbs(name) || filepath.Base(name) != name || strings.ContainsAny(name, `/\`) {
		return "", ErrInvalidName
	}
	base := strings.ToUpper(strings.TrimSuffix(name, filepath.Ext(name)))
	reserved := map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
		"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
		"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
	}
	if reserved[base] || strings.HasSuffix(name, " ") || strings.HasSuffix(name, ".") {
		return "", ErrInvalidName
	}
	return name, nil
}

func scopeFromContext(ctx context.Context) (uint64, string, error) {
	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return 0, "", errors.New("tenant id is required")
	}
	userID, ok := types.UserIDFromContext(ctx)
	if !ok || strings.TrimSpace(userID) == "" {
		return 0, "", errors.New("user id is required")
	}
	return tenantID, userID, nil
}

func publicEntry(stored diskEntry, rootPath, status string, includeRoot bool) *Entry {
	entry := &Entry{
		ID: stored.ID, Name: stored.Name, RelativePath: stored.RelativePath,
		Status: status, TenantID: stored.TenantID, UserID: stored.UserID,
	}
	if includeRoot {
		entry.RootPath = rootPath
	}
	return entry
}

func isWithinRoot(root, target string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(target))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func detectFileKind(name string, isDir bool) string {
	if isDir {
		return "directory"
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".markdown":
		return "markdown"
	case ".txt", ".log", ".json", ".csv", ".xml", ".yaml", ".yml":
		return "text"
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".bmp":
		return "image"
	case ".pdf":
		return "pdf"
	case ".xls", ".xlsx":
		return "spreadsheet"
	case ".ppt", ".pptx", ".key":
		return "presentation"
	case ".zip", ".tar", ".gz", ".7z", ".rar":
		return "archive"
	default:
		return "other"
	}
}

func detectContentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown":
		return "text/markdown; charset=utf-8"
	case ".txt", ".log":
		return "text/plain; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".csv":
		return "text/csv; charset=utf-8"
	case ".xml":
		return "application/xml; charset=utf-8"
	case ".yaml", ".yml":
		return "application/yaml; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	}
	if contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path))); contentType != "" {
		return contentType
	}
	return "application/octet-stream"
}

func isTextPreviewType(path string, contentType string) bool {
	kind := detectFileKind(path, false)
	if kind == "markdown" || kind == "text" {
		return true
	}
	return strings.HasPrefix(contentType, "text/") ||
		strings.HasPrefix(contentType, "application/json") ||
		strings.HasPrefix(contentType, "application/xml") ||
		strings.HasPrefix(contentType, "application/yaml")
}

func probeWritable(root string) error {
	probe := filepath.Join(root, ".xelora-write-probe-"+uuid.NewString())
	file, err := os.OpenFile(probe, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAccessDenied, err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(probe)
		return fmt.Errorf("%w: %v", ErrAccessDenied, err)
	}
	if err := os.Remove(probe); err != nil {
		return fmt.Errorf("%w: %v", ErrAccessDenied, err)
	}
	return nil
}
