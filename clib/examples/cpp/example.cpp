/**
 * LicenseOS C++ example
 *
 * Compile (Linux):
 *   g++ -std=c++17 example.cpp -I../../include -L../../dist/linux \
 *       -llicenseos -Wl,-rpath,'$ORIGIN' -o myapp
 *
 * Compile (Windows, MinGW):
 *   g++ -std=c++17 example.cpp -I../../include -L../../dist/windows \
 *       -llicenseos -o myapp.exe
 */

#include <cstdlib>
#include <iostream>
#include <string>
#include "licenseos.h"

int main()
{
    // ── 1. Create client ──────────────────────────────────────────────────────
    licenseos_handle_t h = LicenseOS_Create(
        "https://license.myapp.com",    // server URL
        "LIC-XXXX-XXXX-XXXX-XXXX",     // license key
        "",                             // CA cert path (optional)
        "",                             // client cert path (mTLS, optional)
        "",                             // client key path  (mTLS, optional)
        ""                              // offline ECDSA public key (optional)
    );

    if (h == LICENSEOS_ERROR) {
        std::cerr << "Failed to create LicenseOS client\n";
        return 1;
    }

    // ── 2. Validate license ───────────────────────────────────────────────────
    char* info_json = LicenseOS_Validate(h);
    if (!info_json) {
        char* err = LicenseOS_GetLastError(h);
        std::cerr << "Validation failed: " << (err ? err : "unknown error") << '\n';
        LicenseOS_Free(err);
        LicenseOS_Destroy(h);
        return 1;
    }
    std::cout << "License info: " << info_json << '\n';
    LicenseOS_Free(info_json);

    // ── 3. Activate machine (idempotent — safe to call every launch) ──────────
    if (LicenseOS_Activate(h) == LICENSEOS_ERROR) {
        char* err = LicenseOS_GetLastError(h);
        std::cerr << "Activation failed: " << (err ? err : "unknown") << '\n';
        LicenseOS_Free(err);
        // non-fatal: continue if already activated
    }

    // ── 4. Open session (enforces concurrent seat limit) ──────────────────────
    if (LicenseOS_StartSession(h) == LICENSEOS_ERROR) {
        char* err = LicenseOS_GetLastError(h);
        std::cerr << "Cannot start session: " << (err ? err : "seat limit reached?") << '\n';
        LicenseOS_Free(err);
        LicenseOS_Destroy(h);
        return 1;
    }
    std::cout << "Session started. Heartbeat is running in background.\n";

    // ── 5. Check features ─────────────────────────────────────────────────────
    if (LicenseOS_HasFeature(h, "export_pdf")) {
        std::cout << "PDF export: ENABLED\n";
    }
    if (LicenseOS_HasFeature(h, "api_access")) {
        std::cout << "API access: ENABLED\n";
    }

    // ── 6. Machine fingerprint ────────────────────────────────────────────────
    char* fp = LicenseOS_GetFingerprint(h);
    std::cout << "Fingerprint: " << (fp ? fp : "n/a") << '\n';
    LicenseOS_Free(fp);

    // ── 7. Run your application ───────────────────────────────────────────────
    std::cout << "Application running...\n";
    // ... your app logic here ...

    // ── 8. Cleanup ────────────────────────────────────────────────────────────
    LicenseOS_StopSession(h);
    LicenseOS_Destroy(h);
    std::cout << "Session released. Goodbye.\n";
    return 0;
}
