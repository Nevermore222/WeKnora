package desktopremote

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

const supportedAPIContractMajor = 1

var ErrIncompatibleAPIContract = errors.New("incompatible API contract")

type SystemCapabilities struct {
	APIContractMajor int      `json:"api_contract_major"`
	APIContractMinor int      `json:"api_contract_minor"`
	ServerVersion    string   `json:"server_version"`
	Features         []string `json:"features"`
}

type Manager struct {
	profiles     ProfileSource
	auth         *AuthClient
	httpClient   *http.Client
	capabilities map[string]SystemCapabilities
}

func NewManager(profiles ProfileSource, auth *AuthClient, httpClient *http.Client) *Manager {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Manager{
		profiles:     profiles,
		auth:         auth,
		httpClient:   httpClient,
		capabilities: map[string]SystemCapabilities{},
	}
}

func ValidateCapabilities(capabilities SystemCapabilities) error {
	if capabilities.APIContractMajor != supportedAPIContractMajor {
		return ErrIncompatibleAPIContract
	}
	return nil
}

func (m *Manager) Activate(ctx context.Context, profileID string) (SystemCapabilities, error) {
	profile, err := m.profiles.Get(ctx, profileID)
	if err != nil {
		return SystemCapabilities{}, err
	}
	target, err := profile.ResolveTarget("/api/v1/system/capabilities")
	if err != nil {
		return SystemCapabilities{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return SystemCapabilities{}, err
	}
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return SystemCapabilities{}, err
	}
	defer resp.Body.Close()

	var capabilities SystemCapabilities
	if err := json.NewDecoder(resp.Body).Decode(&capabilities); err != nil {
		return SystemCapabilities{}, err
	}
	if err := ValidateCapabilities(capabilities); err != nil {
		return SystemCapabilities{}, err
	}
	m.capabilities[profileID] = capabilities
	return capabilities, nil
}
