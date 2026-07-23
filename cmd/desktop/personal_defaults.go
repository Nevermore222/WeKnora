package main

import (
	"os"
	"strings"
)

// applyPersonalDefaults injects sensible default environment variables for the
// personal desktop edition. These are only applied when the variable is not
// already set (by .env, .env.personal, or the OS environment), so users can
// always override them.
func applyPersonalDefaults() {
	defaults := map[string]string{
		"DB_DRIVER":            "sqlite",
		"RETRIEVE_DRIVER":      "sqlite",
		"STORAGE_TYPE":         "local",
		"STREAM_MANAGER_TYPE":  "memory",
		"GIN_MODE":             "release",
		"DISABLE_REGISTRATION": "false",
		"XELORA_SANDBOX_MODE":  "local",
	}
	for key, val := range defaults {
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

func defaultEnterpriseServerURL() string {
	value := strings.TrimSpace(os.Getenv("XELORA_PERSONAL_SERVER_URL"))
	if value == "" {
		value = "http://localhost:8080"
	}
	return strings.TrimRight(value, "/")
}

func defaultEnterpriseServerName() string {
	name := strings.TrimSpace(os.Getenv("XELORA_PERSONAL_SERVER_NAME"))
	if name == "" {
		return "Xelora Server"
	}
	return name
}

func defaultEnterpriseAllowInsecure() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("XELORA_PERSONAL_SERVER_ALLOW_INSECURE")))
	return value == "1" || value == "true" || value == "yes"
}
