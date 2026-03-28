package licenseos

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"runtime"
	"sort"
	"strings"
)

// GenerateFingerprint creates a stable hardware fingerprint from:
// hostname, non-loopback MAC addresses, OS, and architecture.
func GenerateFingerprint() (string, error) {
	var components []string

	if hostname, err := os.Hostname(); err == nil {
		components = append(components, "host:"+hostname)
	}

	macs := macAddresses()
	sort.Strings(macs)
	for _, mac := range macs {
		components = append(components, "mac:"+mac)
	}

	components = append(components, "os:"+runtime.GOOS)
	components = append(components, "arch:"+runtime.GOARCH)

	if len(components) == 0 {
		return "", fmt.Errorf("could not gather any hardware identifiers")
	}

	h := sha256.Sum256([]byte(strings.Join(components, "|")))
	return hex.EncodeToString(h[:]), nil
}

func macAddresses() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var macs []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		mac := iface.HardwareAddr.String()
		if mac != "" && mac != "00:00:00:00:00:00" {
			macs = append(macs, mac)
		}
	}
	return macs
}

// OSInfo returns a string describing the current OS and architecture.
func OSInfo() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}
