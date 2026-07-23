package enterprise

import (
	"context"
	"fmt"
	"time"

	"github.com/Tencent/Xelora/internal/logger"
	"github.com/grandcat/zeroconf"
)

const (
	// xeloraServiceType is the mDNS service type advertised by Xelora servers.
	// Servers that want to be discoverable should register a service of this type.
	xeloraServiceType = "_xelora._tcp"
	defaultBrowseTimeout = 5 * time.Second
)

// DiscoveredServer represents a Xelora server found via mDNS on the LAN.
type DiscoveredServer struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	BaseURL  string `json:"base_url"`
	Instance string `json:"instance"`
}

// DiscoverServers browses the local network for Xelora servers advertising
// the _xelora._tcp mDNS service. Returns whatever is found within the timeout.
// This is best-effort: an empty result is not an error (mDNS may be blocked
// by network policy, or no servers may be advertising).
func DiscoverServers(ctx context.Context) ([]DiscoveredServer, error) {
	entries := make(chan *zeroconf.ServiceEntry, 16)
	var results []DiscoveredServer

	// Collect entries as they arrive.
	done := make(chan struct{})
	go func() {
		for entry := range entries {
			if len(entry.AddrIPv4) == 0 {
				continue
			}
			host := entry.AddrIPv4[0].String()
			results = append(results, DiscoveredServer{
				Name:     entry.Instance,
				Host:     host,
				Port:     entry.Port,
				BaseURL:  fmt.Sprintf("http://%s:%d", host, entry.Port),
				Instance: entry.Instance,
			})
		}
		close(done)
	}()

	browseCtx, cancel := context.WithTimeout(ctx, defaultBrowseTimeout)
	defer cancel()

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("mdns resolver init failed: %w", err)
	}

	if err := resolver.Browse(browseCtx, xeloraServiceType, "local.", entries); err != nil {
		logger.Warnf(ctx, "enterprise: mdns browse failed: %v", err)
		return nil, nil
	}

	<-done
	logger.Infof(ctx, "enterprise: mdns discovery found %d server(s)", len(results))
	return results, nil
}
