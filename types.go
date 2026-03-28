// Package licenseos provides a Go SDK for integrating with the LicenseOS license server.
package licenseos

import "time"

// Config holds the configuration for a LicenseOS client.
type Config struct {
	// ServerURL is the base URL of the LicenseOS server (e.g. "https://license.myapp.com")
	ServerURL string
	// LicenseKey is the license key issued by LicenseOS (e.g. "LIC-XXXX-XXXX-XXXX-XXXX")
	LicenseKey string
	// CertPath is the path to the client certificate PEM file (for mTLS)
	CertPath string
	// KeyPath is the path to the client private key PEM file (for mTLS)
	KeyPath string
	// CACertPath is the path to the CA certificate PEM file
	CACertPath string
	// CachePath is an optional path to cache the offline validation token
	CachePath string
	// OfflinePublicKey is the PEM-encoded ECDSA public key used to verify offline tokens.
	// Obtain it from your LicenseOS server: GET /v1/offline-pubkey
	// Embed as a string constant in your app to enable offline validation.
	// If empty, offline validation is disabled (safer default).
	OfflinePublicKey string
	// Logger is an optional custom logger; uses stdlib log if nil
	Logger Logger
}

// Logger is an optional interface for custom logging.
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// LicenseInfo is returned by Validate().
type LicenseInfo struct {
	LicenseID    string                 `json:"license_id"`
	LicenseKey   string                 `json:"license_key"`
	Status       string                 `json:"status"`
	PlanName     string                 `json:"plan_name"`
	Features     map[string]interface{} `json:"features"`
	ExpiresAt    *time.Time             `json:"expires_at"`
	MaxSeats     *int                   `json:"max_seats"`
	MaxMachines  int                    `json:"max_machines"`
	GraceDays    int                    `json:"offline_grace_days"`
	HeartbeatSec int                    `json:"heartbeat_interval_sec"`
	OfflineToken string                 `json:"offline_token,omitempty"`
}

// Session represents an active license session.
type Session struct {
	ID           string    `json:"session_id"`
	SessionToken string    `json:"session_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// OfflineToken is the parsed payload from the offline JWT token.
type OfflineToken struct {
	LicenseID          string                 `json:"license_id"`
	Features           map[string]interface{} `json:"features"`
	MachineFingerprint string                 `json:"machine_fingerprint"`
	IssuedAt           time.Time              `json:"issued_at"`
	GraceExpiresAt     time.Time              `json:"grace_expires_at"`
}
