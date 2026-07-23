package handler

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

const (
	DesktopAPIContractMajor = 1
	DesktopAPIContractMinor = 0
)

type SystemCapabilitiesResponse struct {
	APIContractMajor int      `json:"api_contract_major"`
	APIContractMinor int      `json:"api_contract_minor"`
	ServerVersion    string   `json:"server_version"`
	Features         []string `json:"features"`
}

func GetSystemCapabilities(c *gin.Context) {
	version := "dev"
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		version = info.Main.Version
	}

	c.JSON(http.StatusOK, SystemCapabilitiesResponse{
		APIContractMajor: DesktopAPIContractMajor,
		APIContractMinor: DesktopAPIContractMinor,
		ServerVersion:    version,
		Features: []string{
			"tenant_rbac",
			"organizations",
			"shared_resources",
			"sse_chat",
		},
	})
}
