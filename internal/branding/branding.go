package branding

import "strings"

// PackagedProfile is injected by the profile-specific release build. Public
// is the source default so existing Xelora builds keep their identity.
var PackagedProfile = "public"

func IsInternal() bool {
	return strings.EqualFold(strings.TrimSpace(PackagedProfile), "internal")
}

func BrandName() string {
	if IsInternal() {
		return "Knavis"
	}
	return "Xelora"
}

func ProductName() string {
	return BrandName() + " Personal"
}
