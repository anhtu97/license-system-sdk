package licenseos

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// validateOffline validates the license using a cached offline token.
func (c *Client) validateOffline() (*LicenseInfo, error) {
	c.mu.RLock()
	token := c.offlineToken
	c.mu.RUnlock()

	if token == nil {
		return nil, fmt.Errorf("no offline token available")
	}
	if time.Now().After(token.GraceExpiresAt) {
		return nil, fmt.Errorf("offline grace period expired at %s", token.GraceExpiresAt.Format(time.RFC3339))
	}
	if token.MachineFingerprint != c.fingerprint {
		return nil, fmt.Errorf("machine fingerprint mismatch")
	}

	return &LicenseInfo{
		LicenseID: token.LicenseID,
		Features:  token.Features,
	}, nil
}

// cacheOfflineToken verifies the JWT signature (ES256) and caches its claims.
// Requires Config.OfflinePublicKey to be set — refuses to cache unverified tokens.
func (c *Client) cacheOfflineToken(tokenStr string) error {
	if c.config.OfflinePublicKey == "" {
		return fmt.Errorf("OfflinePublicKey not configured: cannot verify offline token signature")
	}

	pubKey, err := parseECPublicKey(c.config.OfflinePublicKey)
	if err != nil {
		return fmt.Errorf("invalid OfflinePublicKey: %w", err)
	}

	claims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return pubKey, nil
	})
	if err != nil {
		return fmt.Errorf("offline token signature verification failed: %w", err)
	}

	token := &OfflineToken{
		LicenseID:          claimString(claims, "license_id"),
		MachineFingerprint: claimString(claims, "machine_fingerprint"),
	}

	if features, ok := claims["features"].(map[string]interface{}); ok {
		token.Features = features
	}
	if s := claimString(claims, "grace_expires_at"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			token.GraceExpiresAt = t
		}
	}
	if s := claimString(claims, "issued_at"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			token.IssuedAt = t
		}
	}

	c.mu.Lock()
	c.offlineToken = token
	c.mu.Unlock()

	if c.config.CachePath != "" {
		data, _ := json.Marshal(token)
		_ = os.WriteFile(c.config.CachePath, data, 0600)
	}
	return nil
}

// parseECPublicKey parses a PEM-encoded ECDSA public key.
func parseECPublicKey(pemStr string) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}
	ecKey, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not an ECDSA public key")
	}
	return ecKey, nil
}

// loadOfflineToken reads a cached offline token from disk.
func (c *Client) loadOfflineToken() error {
	if c.config.CachePath == "" {
		return nil
	}
	data, err := os.ReadFile(c.config.CachePath)
	if err != nil {
		return err
	}
	var token OfflineToken
	if err := json.Unmarshal(data, &token); err != nil {
		return err
	}
	c.offlineToken = &token
	return nil
}

func claimString(claims jwt.MapClaims, key string) string {
	if v, ok := claims[key].(string); ok {
		return v
	}
	return ""
}
