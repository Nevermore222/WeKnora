package types

import (
	"strings"
)

// APIKeyScopeType and APIKeyCapability are the small scope vocabulary used by
// the built-in MCP endpoint. Xelora does not yet expose the upstream tenant
// API-key subsystem, so the MCP adapter keeps this projection local instead
// of importing that unrelated feature wholesale.
type APIKeyScopeType string

const APIKeyScopeTenant APIKeyScopeType = "tenant"

type APIKeyCapability string

const (
	APIKeyCapabilityRetrieve   APIKeyCapability = "retrieve"
	APIKeyCapabilityChat       APIKeyCapability = "chat"
	APIKeyCapabilityReadAgents APIKeyCapability = "read_agents"
	APIKeyCapabilityIngest     APIKeyCapability = "ingest"
)

// TenantAPIKeyScope is the normalized authorization projection attached to an
// MCP request. It is intentionally independent from persisted API keys.
type TenantAPIKeyScope struct {
	ScopeType        APIKeyScopeType
	FullAccess       bool
	KnowledgeBaseIDs StringArray
	Capabilities     StringArray
}

func (s TenantAPIKeyScope) HasCapability(capability APIKeyCapability) bool {
	for _, item := range s.Capabilities {
		if strings.TrimSpace(item) == string(capability) {
			return true
		}
	}
	return false
}

func (s TenantAPIKeyScope) AllowsKnowledgeBase(kbID string) bool {
	kbID = strings.TrimSpace(kbID)
	if kbID == "" {
		return false
	}
	if len(s.KnowledgeBaseIDs) == 0 {
		return true
	}
	for _, allowed := range s.KnowledgeBaseIDs {
		if strings.TrimSpace(allowed) == kbID {
			return true
		}
	}
	return false
}
