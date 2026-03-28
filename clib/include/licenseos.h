/**
 * LicenseOS Native SDK
 *
 * C/C++ header for the LicenseOS shared library.
 * Works with: liblicenseos.so (Linux), liblicenseos.dylib (macOS), licenseos.dll (Windows)
 *
 * Usage pattern:
 *   1. LicenseOS_Create(...)      → get handle
 *   2. LicenseOS_Validate(handle) → check license
 *   3. LicenseOS_Activate(handle) → register machine (first run)
 *   4. LicenseOS_StartSession(handle) → open session
 *   5. ... use your app ...
 *   6. LicenseOS_StopSession(handle)
 *   7. LicenseOS_Destroy(handle)
 *
 * Memory rule:
 *   Every function that returns a char* transfers ownership to the caller.
 *   You MUST call LicenseOS_Free(ptr) when done with any returned string.
 */

#ifndef LICENSEOS_H
#define LICENSEOS_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>

#ifdef _WIN32
#  ifdef LICENSEOS_EXPORTS
#    define LICENSEOS_API __declspec(dllexport)
#  else
#    define LICENSEOS_API __declspec(dllimport)
#  endif
#else
#  define LICENSEOS_API
#endif

/** Opaque client handle. A value of LICENSEOS_ERROR indicates failure. */
typedef int licenseos_handle_t;

/** Returned by functions that failed. Check LicenseOS_GetLastError() for details. */
#define LICENSEOS_ERROR (-1)

/**
 * Create a new LicenseOS client.
 *
 * @param server_url         Base URL of the license server, e.g. "https://license.myapp.com"
 * @param license_key        The customer's license key, e.g. "LIC-XXXX-XXXX-XXXX-XXXX"
 * @param ca_cert_path       Path to the CA certificate PEM file, or "" to skip
 * @param cert_path          Path to the client certificate PEM (for mTLS), or ""
 * @param key_path           Path to the client private key PEM (for mTLS), or ""
 * @param offline_public_key PEM-encoded ECDSA P-256 public key for offline validation, or ""
 *
 * @return Handle >= 0 on success, LICENSEOS_ERROR on failure.
 */
LICENSEOS_API licenseos_handle_t LicenseOS_Create(
    const char* server_url,
    const char* license_key,
    const char* ca_cert_path,
    const char* cert_path,
    const char* key_path,
    const char* offline_public_key
);

/**
 * Destroy a client, stop the heartbeat, and release the current session.
 * The handle is invalid after this call.
 */
LICENSEOS_API void LicenseOS_Destroy(licenseos_handle_t handle);

/**
 * Validate the license against the server.
 * Falls back to the cached offline token when the server is unreachable.
 *
 * @return JSON string on success (see below), NULL on error.
 *         Call LicenseOS_GetLastError() for the error message.
 *         Caller MUST free the returned string with LicenseOS_Free().
 *
 * JSON shape:
 * {
 *   "license_id":   "3f2a...",
 *   "license_key":  "LIC-XXXX-XXXX-XXXX-XXXX",
 *   "status":       "active",          // active | trial | expired | suspended
 *   "plan_name":    "Pro",
 *   "expires_at":   "2027-01-01T00:00:00Z",  // omitted for perpetual licenses
 *   "max_seats":    5,
 *   "max_machines": 3,
 *   "grace_days":   7,
 *   "features":     { "api_access": true, "export_pdf": true }
 * }
 */
LICENSEOS_API char* LicenseOS_Validate(licenseos_handle_t handle);

/**
 * Register this machine with the license server.
 * Call once per machine on first launch. Safe to call on every launch.
 *
 * @return 0 on success, LICENSEOS_ERROR on failure.
 */
LICENSEOS_API int LicenseOS_Activate(licenseos_handle_t handle);

/**
 * Open a concurrent-use session and start the automatic heartbeat goroutine.
 * The server enforces max_seats; this call fails when the limit is reached.
 *
 * @return 0 on success, LICENSEOS_ERROR on failure.
 */
LICENSEOS_API int LicenseOS_StartSession(licenseos_handle_t handle);

/**
 * Release the current session and stop the heartbeat goroutine.
 * Call this before LicenseOS_Destroy or when the user exits.
 *
 * @return 0 on success, LICENSEOS_ERROR on failure.
 */
LICENSEOS_API int LicenseOS_StopSession(licenseos_handle_t handle);

/**
 * Check whether a named feature flag is enabled.
 *
 * @param feature_name  Name of the feature, e.g. "api_access"
 * @return 1 if enabled, 0 if disabled or not found.
 */
LICENSEOS_API int LicenseOS_HasFeature(licenseos_handle_t handle, const char* feature_name);

/**
 * Get the raw value of a feature flag as a JSON-encoded string.
 * For boolean features: "true" or "false".
 * For numeric features: "42".
 * For string features: "\"premium\"".
 *
 * @return JSON string, or NULL if the feature does not exist.
 *         Caller MUST free with LicenseOS_Free().
 */
LICENSEOS_API char* LicenseOS_FeatureValue(licenseos_handle_t handle, const char* feature_name);

/**
 * Get this machine's hardware fingerprint (64-character lowercase hex SHA-256).
 *
 * @return Fingerprint string. Caller MUST free with LicenseOS_Free().
 */
LICENSEOS_API char* LicenseOS_GetFingerprint(licenseos_handle_t handle);

/**
 * Retrieve and clear the last error message for this handle.
 *
 * @return Error message string, or NULL if no error occurred since the last call.
 *         Caller MUST free with LicenseOS_Free().
 */
LICENSEOS_API char* LicenseOS_GetLastError(licenseos_handle_t handle);

/**
 * Report a usage metric to the server (for usage-based licensing).
 *
 * @param metric_name  Name of the metric, e.g. "api_calls"
 * @param value        Metric value
 * @return 0 on success, LICENSEOS_ERROR on failure.
 */
LICENSEOS_API int LicenseOS_ReportUsage(licenseos_handle_t handle, const char* metric_name, int64_t value);

/**
 * Free a string returned by any LicenseOS_* function.
 * Equivalent to calling free() on the pointer.
 * Passing NULL is safe (no-op).
 */
LICENSEOS_API void LicenseOS_Free(char* ptr);

#ifdef __cplusplus
} /* extern "C" */
#endif

#endif /* LICENSEOS_H */
