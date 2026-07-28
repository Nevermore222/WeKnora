package types

// AdminUserListFilter selects which users appear in the system-admin
// user-permissions list. Empty or "all" means no flag filter.
type AdminUserListFilter string

const (
	AdminUserFilterAll         AdminUserListFilter = "all"
	AdminUserFilterClient      AdminUserListFilter = "client"
	AdminUserFilterNonClient   AdminUserListFilter = "non_client"
	AdminUserFilterSystemAdmin AdminUserListFilter = "system_admin"
)

// AdminUserListItem is one row for GET /system/admin/users.
type AdminUserListItem struct {
	User           *User
	HomeTenantName string
}
