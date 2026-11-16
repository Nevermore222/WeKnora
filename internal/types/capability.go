package types

// Capability identifiers used by dual-gate authorization (tenant RBAC AND
// per-user capability). Keep IDs identical to User JSON field names so
// frontend /auth/me flags and middleware share one vocabulary.
const (
	CapabilityCreateKnowledgeBase = "can_create_knowledge_base"
	CapabilityCreateAgent         = "can_create_agent"
	CapabilityBindModel           = "can_bind_model"
)

// CapabilityDef documents a registered capability for admin UI and future
// extensions. Defaults for client users live in system settings, not here.
type CapabilityDef struct {
	ID string
}

// CapabilityRegistry returns the v1 capability list. New capabilities should
// be appended here and mirrored on User / UserInfo + migration columns.
func CapabilityRegistry() []CapabilityDef {
	return []CapabilityDef{
		{ID: CapabilityCreateKnowledgeBase},
		{ID: CapabilityCreateAgent},
		{ID: CapabilityBindModel},
	}
}

// UserHasCapability reports whether u is allowed the given capability id.
// System administrators have every registered capability. Nil users and
// unknown ids deny.
func UserHasCapability(u *User, id string) bool {
	if u == nil {
		return false
	}
	if u.IsSystemAdmin {
		for _, capability := range CapabilityRegistry() {
			if capability.ID == id {
				return true
			}
		}
		return false
	}
	switch id {
	case CapabilityCreateKnowledgeBase:
		return u.CanCreateKnowledgeBase
	case CapabilityCreateAgent:
		return u.CanCreateAgent
	case CapabilityBindModel:
		return u.CanBindModel
	default:
		return false
	}
}
