package desktopremote

import (
	"errors"
	"net"
	"net/url"
	"strings"
	"time"
)

type ServerProfile struct {
	ID                            string    `gorm:"primaryKey;size:36" json:"id"`
	Name                          string    `gorm:"not null" json:"name"`
	BaseURL                       string    `gorm:"not null" json:"base_url"`
	AllowInsecureTransport        bool      `gorm:"not null;default:false" json:"allow_insecure_transport"`
	TrustedCertificateFingerprint string    `json:"trusted_certificate_fingerprint,omitempty"`
	LastUserID                    string    `json:"last_user_id,omitempty"`
	LastTenantID                  uint64    `json:"last_tenant_id,omitempty"`
	CreatedAt                     time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt                     time.Time `gorm:"not null" json:"updated_at"`
}

func (ServerProfile) TableName() string {
	return "desktop_server_profiles"
}

func NormalizeServerOrigin(raw string, allowInsecure bool) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || u.Opaque != "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("server address must be an http(s) origin")
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", errors.New("server address must be an http(s) origin")
	}
	if u.Path != "" && u.Path != "/" {
		return "", errors.New("server address must not contain a path")
	}

	hostname := strings.ToLower(u.Hostname())
	ip := net.ParseIP(hostname)
	isLoopback := hostname == "localhost" || ip != nil && ip.IsLoopback()
	if scheme != "https" && !(scheme == "http" && (isLoopback || allowInsecure)) {
		return "", errors.New("HTTPS is required")
	}

	host := hostname
	if strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}
	if port := u.Port(); port != "" {
		host += ":" + port
	}

	return (&url.URL{Scheme: scheme, Host: host}).String(), nil
}

func (p ServerProfile) ResolveTarget(rawPath string) (*url.URL, error) {
	origin, err := NormalizeServerOrigin(p.BaseURL, p.AllowInsecureTransport)
	if err != nil {
		return nil, err
	}
	relative, err := url.Parse(rawPath)
	if err != nil || relative.IsAbs() || relative.Host != "" || relative.User != nil || relative.Fragment != "" || !strings.HasPrefix(relative.Path, "/") {
		return nil, errors.New("target must be an absolute-path reference on the configured server")
	}

	base, err := url.Parse(origin)
	if err != nil {
		return nil, err
	}
	target := base.ResolveReference(relative)
	if target.Scheme != base.Scheme || target.Host != base.Host {
		return nil, errors.New("target origin does not match configured server")
	}
	return target, nil
}
