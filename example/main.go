package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	licenseos "github.com/licenseos/sdk"
)

func main() {
	// Initialize the LicenseOS client.
	// For development (no mTLS), set only ServerURL + LicenseKey.
	client, err := licenseos.New(licenseos.Config{
		ServerURL:  getEnv("LICENSE_SERVER_URL", "http://localhost:8080"),
		LicenseKey: getEnv("LICENSE_KEY", "LIC-DEMO-PRO0-0001-ACME"),
		// For production with mTLS:
		// CertPath:   "/etc/myapp/client.crt",
		// KeyPath:    "/etc/myapp/client.key",
		// CACertPath: "/etc/myapp/ca.crt",
		CachePath: "/tmp/licenseos.cache",
	})
	if err != nil {
		log.Fatalf("Failed to initialize LicenseOS client: %v", err)
	}

	// Register this machine (run on first start)
	if err := client.Activate(); err != nil {
		log.Printf("Machine activation warning: %v", err)
	}

	// Validate license (falls back to offline token on network failure)
	info, err := client.Validate()
	if err != nil {
		log.Fatalf("License validation failed: %v\nObtain a valid license at https://licenseos.example.com", err)
	}

	fmt.Printf("✓ License valid\n")
	fmt.Printf("  Plan:    %s\n", info.PlanName)
	fmt.Printf("  Status:  %s\n", info.Status)
	if info.ExpiresAt != nil {
		fmt.Printf("  Expires: %s\n", info.ExpiresAt.Format("2006-01-02"))
	}

	// Start session (auto-heartbeat in background)
	session, err := client.StartSession()
	if err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}
	fmt.Printf("  Session: %s (expires %s)\n\n", session.ID, session.ExpiresAt.Format(time.RFC3339))

	// Release session on exit
	defer func() {
		fmt.Println("Releasing session...")
		if err := client.StopSession(); err != nil {
			log.Printf("Failed to stop session: %v", err)
		}
	}()

	// Gate features
	if client.HasFeature("reporting") {
		fmt.Println("✓ Reporting module enabled")
	} else {
		fmt.Println("✗ Reporting module disabled (upgrade to Pro)")
	}

	if apiLimit := client.FeatureValue("api_limit"); apiLimit != nil {
		fmt.Printf("✓ API limit: %.0f calls/month\n", apiLimit.(float64))
	}

	// Report usage
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\nApplication running (Ctrl+C to exit)...")
	counter := int64(0)
	for {
		select {
		case <-ticker.C:
			counter++
			if err := client.ReportUsage("api_calls", counter); err != nil {
				log.Printf("Usage report error: %v", err)
			} else {
				fmt.Printf("  Reported api_calls = %d\n", counter)
			}
		case <-stop:
			fmt.Println("Shutting down...")
			return
		}
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
