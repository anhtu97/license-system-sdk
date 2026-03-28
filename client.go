package licenseos

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// Client is the LicenseOS SDK client.
// Create one with New() and call Validate() before using the application.
type Client struct {
	config      Config
	httpClient  *http.Client

	mu          sync.RWMutex
	licenseInfo *LicenseInfo
	session     *Session
	fingerprint string

	heartbeatStop chan struct{}
	heartbeatDone chan struct{}

	offlineToken *OfflineToken
}

// New creates and initialises a LicenseOS client.
// It loads mTLS certificates when CertPath / KeyPath / CACertPath are set.
func New(cfg Config) (*Client, error) {
	fp, err := GenerateFingerprint()
	if err != nil {
		return nil, fmt.Errorf("fingerprint: %w", err)
	}

	c := &Client{
		config:      cfg,
		fingerprint: fp,
	}

	httpClient, err := c.buildHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("http client: %w", err)
	}
	c.httpClient = httpClient

	_ = c.loadOfflineToken() // best-effort

	return c, nil
}

func (c *Client) buildHTTPClient() (*http.Client, error) {
	tlsCfg := &tls.Config{}

	if c.config.CertPath != "" && c.config.KeyPath != "" {
		cert, err := tls.LoadX509KeyPair(c.config.CertPath, c.config.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("load client cert: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	if c.config.CACertPath != "" {
		pem, err := os.ReadFile(c.config.CACertPath)
		if err != nil {
			return nil, fmt.Errorf("read CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("parse CA cert")
		}
		tlsCfg.RootCAs = pool
	}

	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{TLSClientConfig: tlsCfg},
	}, nil
}

// Validate validates the license key against the server.
// On network failure it falls back to the cached offline token.
func (c *Client) Validate() (*LicenseInfo, error) {
	payload := map[string]string{
		"license_key":         c.config.LicenseKey,
		"machine_fingerprint": c.fingerprint,
		"client_time":         time.Now().UTC().Format(time.RFC3339),
	}

	var info LicenseInfo
	if err := c.post("/v1/licenses/validate", payload, &info); err != nil {
		c.logf("online validation failed (%v), trying offline", err)
		return c.validateOffline()
	}

	c.mu.Lock()
	c.licenseInfo = &info
	c.mu.Unlock()

	if info.OfflineToken != "" {
		_ = c.cacheOfflineToken(info.OfflineToken)
	}
	return &info, nil
}

// Activate registers this machine with the license server.
func (c *Client) Activate() error {
	hostname, _ := os.Hostname()
	payload := map[string]string{
		"license_key":         c.config.LicenseKey,
		"machine_fingerprint": c.fingerprint,
		"hostname":            hostname,
		"os_info":             OSInfo(),
	}
	return c.post("/v1/licenses/activate", payload, nil)
}

// StartSession creates a new session lease and begins automatic heartbeats.
func (c *Client) StartSession() (*Session, error) {
	heartbeatSec := 60
	c.mu.RLock()
	if c.licenseInfo != nil && c.licenseInfo.HeartbeatSec > 0 {
		heartbeatSec = c.licenseInfo.HeartbeatSec
	}
	c.mu.RUnlock()

	payload := map[string]interface{}{
		"license_key":         c.config.LicenseKey,
		"machine_fingerprint": c.fingerprint,
		"lease_duration_sec":  heartbeatSec * 2,
	}

	var session Session
	if err := c.post("/v1/sessions/create", payload, &session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	c.mu.Lock()
	c.session = &session
	c.heartbeatStop = make(chan struct{})
	c.heartbeatDone = make(chan struct{})
	c.mu.Unlock()

	go c.runHeartbeat(time.Duration(heartbeatSec) * time.Second)
	return &session, nil
}

// StopSession releases the current session and stops the heartbeat goroutine.
func (c *Client) StopSession() error {
	c.mu.Lock()
	stop := c.heartbeatStop
	session := c.session
	c.heartbeatStop = nil
	c.mu.Unlock()

	if stop != nil {
		close(stop)
		<-c.heartbeatDone
	}
	if session == nil {
		return nil
	}
	return c.post("/v1/sessions/release", map[string]string{"session_token": session.SessionToken}, nil)
}

func (c *Client) runHeartbeat(interval time.Duration) {
	defer close(c.heartbeatDone)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.RLock()
			session := c.session
			c.mu.RUnlock()
			if session == nil {
				return
			}
			var resp struct {
				ExpiresAt time.Time `json:"expires_at"`
			}
			if err := c.post("/v1/sessions/heartbeat",
				map[string]string{"session_token": session.SessionToken}, &resp); err != nil {
				c.logf("heartbeat error: %v", err)
				continue
			}
			c.mu.Lock()
			if c.session != nil {
				c.session.ExpiresAt = resp.ExpiresAt
			}
			c.mu.Unlock()

		case <-c.heartbeatStop:
			return
		}
	}
}

// HasFeature returns true if the named feature is enabled.
func (c *Client) HasFeature(name string) bool {
	features := c.features()
	val, ok := features[name]
	if !ok {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case float64:
		return v > 0
	case string:
		return v != "" && v != "false" && v != "0"
	}
	return val != nil
}

// FeatureValue returns the raw value of a feature flag.
func (c *Client) FeatureValue(name string) interface{} {
	return c.features()[name]
}

func (c *Client) features() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.licenseInfo != nil {
		return c.licenseInfo.Features
	}
	if c.offlineToken != nil {
		return c.offlineToken.Features
	}
	return map[string]interface{}{}
}

// ReportUsage reports a usage metric to the server.
func (c *Client) ReportUsage(metric string, value int64) error {
	payload := map[string]interface{}{
		"license_key":         c.config.LicenseKey,
		"machine_fingerprint": c.fingerprint,
		"metric_name":         metric,
		"metric_value":        value,
	}
	return c.post("/v1/usage/report", payload, nil)
}

// GetFingerprint returns this machine's hardware fingerprint.
func (c *Client) GetFingerprint() string {
	return c.fingerprint
}

// post sends a JSON POST request to path and decodes the response into out (if non-nil).
func (c *Client) post(path string, body interface{}, out interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.config.ServerURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.config.LicenseKey != "" {
		req.Header.Set("X-License-Key", c.config.LicenseKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		var e struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(respData, &e)
		msg := e.Error
		if msg == "" {
			msg = e.Message
		}
		if msg == "" {
			msg = string(respData)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
	}
	if out != nil {
		return json.Unmarshal(respData, out)
	}
	return nil
}

func (c *Client) logf(format string, args ...interface{}) {
	if c.config.Logger != nil {
		c.config.Logger.Errorf(format, args...)
	} else {
		log.Printf("[licenseos] "+format, args...)
	}
}
