package main

import (
	"fmt"
	"strings"
)

func ExtractSubdomain(host, domain string) (string, error) {
	// Strip port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		h := host[:idx]
		// Only strip if what's after : looks like a port (not part of IPv6)
		if !strings.Contains(h, ":") {
			host = h
		}
	}

	suffix := "." + domain
	if !strings.HasSuffix(host, suffix) {
		return "", fmt.Errorf("host %q does not match domain %q", host, domain)
	}

	subdomain := strings.TrimSuffix(host, suffix)
	if subdomain == "" {
		return "", fmt.Errorf("no subdomain in host %q", host)
	}

	return subdomain, nil
}
