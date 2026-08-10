package container

import (
	"fmt"
	"os"
	"strings"
)

const processRoleEnv = "XELORA_PROCESS_ROLE"

type processRole string

const (
	processRoleActive  processRole = "active"
	processRolePassive processRole = "passive"
)

func processRoleFromEnv() (processRole, error) {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(processRoleEnv))) {
	case "", string(processRoleActive):
		return processRoleActive, nil
	case string(processRolePassive):
		return processRolePassive, nil
	default:
		return "", fmt.Errorf("%s must be active or passive", processRoleEnv)
	}
}

func backgroundServicesEnabled(role processRole) bool {
	return role == processRoleActive
}
