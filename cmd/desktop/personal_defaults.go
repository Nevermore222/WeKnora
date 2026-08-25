package main

import (
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/Tencent/Xelora/internal/branding"
)

// packagedEnterpriseServerURL is injected into internal desktop builds with
// Go's -ldflags -X. An empty value preserves the developer fallback below.
var packagedEnterpriseServerURL string

// packagedDesktopUpdateFeedURL is injected into internal desktop builds so
// update checks use the frontend-hosted Knavis release channel rather than the
// backend API port or the public Xelora release feed.
var packagedDesktopUpdateFeedURL string

func desktopBrandProfileIsInternal() bool {
	return branding.IsInternal()
}

func desktopBrandName() string {
	return branding.BrandName()
}

func desktopProductName() string {
	return branding.ProductName()
}

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
	value := strings.TrimSpace(packagedEnterpriseServerURL)
	if value == "" {
		value = strings.TrimSpace(os.Getenv("XELORA_PERSONAL_SERVER_URL"))
	}
	if value == "" {
		value = "http://localhost:8080"
	}
	return strings.TrimRight(value, "/")
}

func defaultEnterpriseServerFallbackURL(primary string) string {
	return localEnterpriseServerFallbackURL(primary, desktopLocalIPv4Addresses())
}

func localEnterpriseServerFallbackURL(primary string, localIPs []net.IP) string {
	u, err := url.Parse(strings.TrimSpace(primary))
	if err != nil || u.Host == "" {
		return ""
	}

	hostIP := net.ParseIP(u.Hostname())
	if hostIP == nil || hostIP.IsLoopback() {
		return ""
	}
	for _, localIP := range localIPs {
		if localIP != nil && localIP.Equal(hostIP) {
			host := "127.0.0.1"
			if port := u.Port(); port != "" {
				host = net.JoinHostPort(host, port)
			}
			u.Host = host
			u.Path = ""
			u.RawPath = ""
			u.RawQuery = ""
			u.Fragment = ""
			return strings.TrimRight(u.String(), "/")
		}
	}
	return ""
}

func desktopLocalIPv4Addresses() []net.IP {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	var localIPs []net.IP
	for _, iface := range interfaces {
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			var ip net.IP
			switch value := address.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}
			if ipv4 := ip.To4(); ipv4 != nil && !ipv4.IsLoopback() {
				localIPs = append(localIPs, ipv4)
			}
		}
	}
	return localIPs
}

func defaultEnterpriseServerName() string {
	name := strings.TrimSpace(os.Getenv("XELORA_PERSONAL_SERVER_NAME"))
	if name == "" {
		return desktopBrandName() + " Server"
	}
	return name
}

func defaultEnterpriseAllowInsecure() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("XELORA_PERSONAL_SERVER_ALLOW_INSECURE")))
	return value == "1" || value == "true" || value == "yes"
}
