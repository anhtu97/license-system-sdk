// Package main exposes the LicenseOS SDK as a C-compatible shared library.
//
// Build targets (see Makefile):
//   Linux:   liblicenseos.so
//   macOS:   liblicenseos.dylib
//   Windows: licenseos.dll
//
// All functions returning *C.char transfer ownership to the caller.
// The caller MUST free those strings with LicenseOS_Free().
package main

/*
#include <stdlib.h>
#include <stdint.h>
*/
import "C"

import (
	"encoding/json"
	"sync"
	"unsafe"

	licenseos "github.com/licenseos/sdk"
)

// ── Handle registry ───────────────────────────────────────────────────────────

var (
	mu      sync.Mutex
	clients = map[C.int]*clientState{}
	nextID  C.int = 1
)

type clientState struct {
	client    *licenseos.Client
	lastError string
}

func acquire(handle C.int) *clientState {
	mu.Lock()
	defer mu.Unlock()
	return clients[handle]
}

func recordError(handle C.int, msg string) {
	mu.Lock()
	defer mu.Unlock()
	if s, ok := clients[handle]; ok {
		s.lastError = msg
	}
}

// ── Exported API ──────────────────────────────────────────────────────────────

// LicenseOS_Create initialises a new SDK client and returns an opaque integer handle.
// Returns LICENSEOS_ERROR (-1) on failure (fingerprint collection or TLS setup failed).
// All string parameters may be empty ("") when not needed.
//
//   server_url        — e.g. "https://license.myapp.com"
//   license_key       — e.g. "LIC-XXXX-XXXX-XXXX-XXXX"
//   ca_cert_path      — path to CA .crt file, or ""
//   cert_path         — path to client .crt for mTLS, or ""
//   key_path          — path to client .key for mTLS, or ""
//   offline_public_key — PEM ECDSA public key for offline validation, or ""
//
//export LicenseOS_Create
func LicenseOS_Create(serverURL, licenseKey, caCertPath, certPath, keyPath, offlinePublicKey *C.char) C.int {
	cfg := licenseos.Config{
		ServerURL:        C.GoString(serverURL),
		LicenseKey:       C.GoString(licenseKey),
		CACertPath:       C.GoString(caCertPath),
		CertPath:         C.GoString(certPath),
		KeyPath:          C.GoString(keyPath),
		OfflinePublicKey: C.GoString(offlinePublicKey),
	}

	client, err := licenseos.New(cfg)
	if err != nil {
		return -1
	}

	mu.Lock()
	id := nextID
	nextID++
	clients[id] = &clientState{client: client}
	mu.Unlock()

	return id
}

// LicenseOS_Destroy stops any active session and releases all resources for handle.
//
//export LicenseOS_Destroy
func LicenseOS_Destroy(handle C.int) {
	mu.Lock()
	s, ok := clients[handle]
	if ok {
		delete(clients, handle)
	}
	mu.Unlock()
	if ok {
		_ = s.client.StopSession()
	}
}

// LicenseOS_Validate checks the license against the server (or falls back to offline token).
// Returns a JSON string on success, NULL on error — check LicenseOS_GetLastError().
// Caller must free the returned string with LicenseOS_Free().
//
// JSON shape:
//
//	{
//	  "license_id":   "uuid",
//	  "license_key":  "LIC-...",
//	  "status":       "active",
//	  "plan_name":    "Pro",
//	  "expires_at":   "2027-01-01T00:00:00Z",  // omitted if perpetual
//	  "max_seats":    5,
//	  "max_machines": 3,
//	  "grace_days":   7,
//	  "features":     { "api_access": true, "export_pdf": true }
//	}
//
//export LicenseOS_Validate
func LicenseOS_Validate(handle C.int) *C.char {
	s := acquire(handle)
	if s == nil {
		return nil
	}

	info, err := s.client.Validate()
	if err != nil {
		recordError(handle, err.Error())
		return nil
	}

	type result struct {
		LicenseID   string      `json:"license_id"`
		LicenseKey  string      `json:"license_key"`
		Status      string      `json:"status"`
		PlanName    string      `json:"plan_name"`
		ExpiresAt   string      `json:"expires_at,omitempty"`
		MaxSeats    int         `json:"max_seats"`
		MaxMachines int         `json:"max_machines"`
		GraceDays   int         `json:"grace_days"`
		Features    interface{} `json:"features"`
	}
	r := result{
		LicenseID:   info.LicenseID,
		LicenseKey:  info.LicenseKey,
		Status:      info.Status,
		PlanName:    info.PlanName,
		MaxMachines: info.MaxMachines,
		GraceDays:   info.GraceDays,
		Features:    info.Features,
	}
	if info.MaxSeats != nil {
		r.MaxSeats = *info.MaxSeats
	}
	if info.ExpiresAt != nil {
		r.ExpiresAt = info.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z")
	}

	b, err := json.Marshal(r)
	if err != nil {
		recordError(handle, err.Error())
		return nil
	}
	return C.CString(string(b))
}

// LicenseOS_Activate registers this machine with the license server.
// Returns 0 on success, -1 on error — check LicenseOS_GetLastError().
//
//export LicenseOS_Activate
func LicenseOS_Activate(handle C.int) C.int {
	s := acquire(handle)
	if s == nil {
		return -1
	}
	if err := s.client.Activate(); err != nil {
		recordError(handle, err.Error())
		return -1
	}
	return 0
}

// LicenseOS_StartSession opens a session lease and starts the heartbeat goroutine.
// Returns 0 on success, -1 on error.
//
//export LicenseOS_StartSession
func LicenseOS_StartSession(handle C.int) C.int {
	s := acquire(handle)
	if s == nil {
		return -1
	}
	if _, err := s.client.StartSession(); err != nil {
		recordError(handle, err.Error())
		return -1
	}
	return 0
}

// LicenseOS_StopSession releases the current session and stops the heartbeat.
// Returns 0 on success, -1 on error.
//
//export LicenseOS_StopSession
func LicenseOS_StopSession(handle C.int) C.int {
	s := acquire(handle)
	if s == nil {
		return -1
	}
	if err := s.client.StopSession(); err != nil {
		recordError(handle, err.Error())
		return -1
	}
	return 0
}

// LicenseOS_HasFeature returns 1 if the named feature flag is enabled, 0 otherwise.
//
//export LicenseOS_HasFeature
func LicenseOS_HasFeature(handle C.int, featureName *C.char) C.int {
	s := acquire(handle)
	if s == nil {
		return 0
	}
	if s.client.HasFeature(C.GoString(featureName)) {
		return 1
	}
	return 0
}

// LicenseOS_FeatureValue returns the raw value of a feature flag serialised as JSON,
// or NULL if the feature does not exist.
// Caller must free the returned string with LicenseOS_Free().
//
//export LicenseOS_FeatureValue
func LicenseOS_FeatureValue(handle C.int, featureName *C.char) *C.char {
	s := acquire(handle)
	if s == nil {
		return nil
	}
	val := s.client.FeatureValue(C.GoString(featureName))
	if val == nil {
		return nil
	}
	b, err := json.Marshal(val)
	if err != nil {
		return nil
	}
	return C.CString(string(b))
}

// LicenseOS_GetFingerprint returns this machine's hardware fingerprint (64-char hex SHA-256).
// Caller must free the returned string with LicenseOS_Free().
//
//export LicenseOS_GetFingerprint
func LicenseOS_GetFingerprint(handle C.int) *C.char {
	s := acquire(handle)
	if s == nil {
		return nil
	}
	return C.CString(s.client.GetFingerprint())
}

// LicenseOS_GetLastError returns the last error message for handle, or NULL if no error.
// Caller must free the returned string with LicenseOS_Free().
//
//export LicenseOS_GetLastError
func LicenseOS_GetLastError(handle C.int) *C.char {
	mu.Lock()
	defer mu.Unlock()
	s, ok := clients[handle]
	if !ok || s.lastError == "" {
		return nil
	}
	msg := s.lastError
	s.lastError = "" // clear after reading
	return C.CString(msg)
}

// LicenseOS_ReportUsage sends a usage metric to the server.
// Returns 0 on success, -1 on error.
//
//export LicenseOS_ReportUsage
func LicenseOS_ReportUsage(handle C.int, metricName *C.char, value C.longlong) C.int {
	s := acquire(handle)
	if s == nil {
		return -1
	}
	if err := s.client.ReportUsage(C.GoString(metricName), int64(value)); err != nil {
		recordError(handle, err.Error())
		return -1
	}
	return 0
}

// LicenseOS_Free releases a string returned by any LicenseOS_* function.
// Always call this when you are done with a returned string.
//
//export LicenseOS_Free
func LicenseOS_Free(ptr *C.char) {
	C.free(unsafe.Pointer(ptr))
}

func main() {}
