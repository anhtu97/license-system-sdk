// LicenseOS .NET SDK — P/Invoke wrapper
//
// Targets: .NET 6+ / .NET Framework 4.7.2+
//
// Place the native library next to your executable:
//   Windows : licenseos.dll
//   Linux   : liblicenseos.so
//   macOS   : liblicenseos.dylib
//
// Usage:
//   using var client = new LicenseOS.Client(
//       serverUrl:  "https://license.myapp.com",
//       licenseKey: "LIC-XXXX-XXXX-XXXX-XXXX"
//   );
//   var info = client.Validate();                // throws on failure
//   client.Activate();
//   client.StartSession();
//   bool hasPdf = client.HasFeature("export_pdf");
//   // ... run your app ...
//   // Dispose() / using statement calls StopSession + Destroy automatically

using System;
using System.Runtime.InteropServices;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace LicenseOS
{
    // ── Result types ──────────────────────────────────────────────────────────

    public sealed class LicenseInfo
    {
        [JsonPropertyName("license_id")]   public string LicenseId   { get; set; } = "";
        [JsonPropertyName("license_key")]  public string LicenseKey  { get; set; } = "";
        [JsonPropertyName("status")]       public string Status      { get; set; } = "";
        [JsonPropertyName("plan_name")]    public string PlanName    { get; set; } = "";
        [JsonPropertyName("expires_at")]   public string? ExpiresAt  { get; set; }
        [JsonPropertyName("max_seats")]    public int MaxSeats       { get; set; }
        [JsonPropertyName("max_machines")] public int MaxMachines    { get; set; }
        [JsonPropertyName("grace_days")]   public int GraceDays      { get; set; }
        [JsonPropertyName("features")]     public JsonElement Features { get; set; }
    }

    public sealed class LicenseOSException : Exception
    {
        public LicenseOSException(string message) : base(message) { }
    }

    // ── Native bindings ───────────────────────────────────────────────────────

    internal static class Native
    {
#if WINDOWS
        private const string Lib = "licenseos";
#elif MACOS
        private const string Lib = "liblicenseos";
#else
        private const string Lib = "liblicenseos";   // Linux default
#endif

        [DllImport(Lib, CallingConvention = CallingConvention.Cdecl)]
        public static extern int LicenseOS_Create(
            [MarshalAs(UnmanagedType.LPUTF8Str)] string serverUrl,
            [MarshalAs(UnmanagedType.LPUTF8Str)] string licenseKey,
            [MarshalAs(UnmanagedType.LPUTF8Str)] string caCertPath,
            [MarshalAs(UnmanagedType.LPUTF8Str)] string certPath,
            [MarshalAs(UnmanagedType.LPUTF8Str)] string keyPath,
            [MarshalAs(UnmanagedType.LPUTF8Str)] string offlinePublicKey);

        [DllImport(Lib, CallingConvention = CallingConvention.Cdecl)]
        public static extern void LicenseOS_Destroy(int handle);

        [DllImport(Lib, CallingConvention = CallingConvention.Cdecl)]
        public static extern IntPtr LicenseOS_Validate(int handle);

        [DllImport(Lib, CallingConvention = CallingConvention.Cdecl)]
        public static extern int LicenseOS_Activate(int handle);

        [DllImport(Lib, CallingConvention = CallingConvention.Cdecl)]
        public static extern int LicenseOS_StartSession(int handle);

        [DllImport(Lib, CallingConvention = CallingConvention.Cdecl)]
        public static extern int LicenseOS_StopSession(int handle);

        [DllImport(Lib, CallingConvention = CallingConvention.Cdecl)]
        public static extern int LicenseOS_HasFeature(
            int handle,
            [MarshalAs(UnmanagedType.LPUTF8Str)] string featureName);

        [DllImport(Lib, CallingConvention = CallingConvention.Cdecl)]
        public static extern IntPtr LicenseOS_FeatureValue(
            int handle,
            [MarshalAs(UnmanagedType.LPUTF8Str)] string featureName);

        [DllImport(Lib, CallingConvention = CallingConvention.Cdecl)]
        public static extern IntPtr LicenseOS_GetFingerprint(int handle);

        [DllImport(Lib, CallingConvention = CallingConvention.Cdecl)]
        public static extern IntPtr LicenseOS_GetLastError(int handle);

        [DllImport(Lib, CallingConvention = CallingConvention.Cdecl)]
        public static extern int LicenseOS_ReportUsage(
            int handle,
            [MarshalAs(UnmanagedType.LPUTF8Str)] string metricName,
            long value);

        [DllImport(Lib, CallingConvention = CallingConvention.Cdecl)]
        public static extern void LicenseOS_Free(IntPtr ptr);
    }

    // ── Public client ─────────────────────────────────────────────────────────

    /// <summary>
    /// LicenseOS client. Dispose to release the session automatically.
    /// </summary>
    public sealed class Client : IDisposable
    {
        private int _handle;
        private bool _disposed;

        /// <param name="serverUrl">Base URL of the LicenseOS server.</param>
        /// <param name="licenseKey">Customer license key.</param>
        /// <param name="caCertPath">Optional path to CA certificate PEM (verify server).</param>
        /// <param name="certPath">Optional path to client certificate PEM (mTLS).</param>
        /// <param name="keyPath">Optional path to client private key PEM (mTLS).</param>
        /// <param name="offlinePublicKey">ECDSA PEM public key for offline validation.</param>
        public Client(
            string serverUrl,
            string licenseKey,
            string caCertPath       = "",
            string certPath         = "",
            string keyPath          = "",
            string offlinePublicKey = "")
        {
            _handle = Native.LicenseOS_Create(
                serverUrl, licenseKey, caCertPath, certPath, keyPath, offlinePublicKey);

            if (_handle == -1)
                throw new LicenseOSException("Failed to initialise LicenseOS client (check fingerprint/TLS config).");
        }

        /// <summary>
        /// Validate the license. Throws <see cref="LicenseOSException"/> on failure.
        /// </summary>
        public LicenseInfo Validate()
        {
            ThrowIfDisposed();
            var ptr = Native.LicenseOS_Validate(_handle);
            if (ptr == IntPtr.Zero)
                throw new LicenseOSException(GetLastError() ?? "Validation failed");

            try
            {
                var json = Marshal.PtrToStringUTF8(ptr)!;
                return JsonSerializer.Deserialize<LicenseInfo>(json)
                    ?? throw new LicenseOSException("Failed to parse validation response.");
            }
            finally
            {
                Native.LicenseOS_Free(ptr);
            }
        }

        /// <summary>
        /// Register this machine with the license server.
        /// Safe to call on every launch; idempotent.
        /// </summary>
        public void Activate()
        {
            ThrowIfDisposed();
            if (Native.LicenseOS_Activate(_handle) == -1)
                throw new LicenseOSException(GetLastError() ?? "Activation failed");
        }

        /// <summary>
        /// Open a concurrent-use session and start the heartbeat.
        /// Throws when the seat limit is reached.
        /// </summary>
        public void StartSession()
        {
            ThrowIfDisposed();
            if (Native.LicenseOS_StartSession(_handle) == -1)
                throw new LicenseOSException(GetLastError() ?? "Failed to start session");
        }

        /// <summary>Stop the current session and heartbeat.</summary>
        public void StopSession()
        {
            ThrowIfDisposed();
            if (Native.LicenseOS_StopSession(_handle) == -1)
                throw new LicenseOSException(GetLastError() ?? "Failed to stop session");
        }

        /// <summary>Returns true if the named feature flag is enabled.</summary>
        public bool HasFeature(string featureName)
        {
            ThrowIfDisposed();
            return Native.LicenseOS_HasFeature(_handle, featureName) == 1;
        }

        /// <summary>
        /// Returns the raw JSON value of a feature flag, or null if not found.
        /// e.g. "true", "42", "\"premium\""
        /// </summary>
        public string? GetFeatureValue(string featureName)
        {
            ThrowIfDisposed();
            var ptr = Native.LicenseOS_FeatureValue(_handle, featureName);
            if (ptr == IntPtr.Zero) return null;
            try   { return Marshal.PtrToStringUTF8(ptr); }
            finally { Native.LicenseOS_Free(ptr); }
        }

        /// <summary>Returns this machine's hardware fingerprint (64-char hex SHA-256).</summary>
        public string GetFingerprint()
        {
            ThrowIfDisposed();
            var ptr = Native.LicenseOS_GetFingerprint(_handle);
            if (ptr == IntPtr.Zero) return "";
            try   { return Marshal.PtrToStringUTF8(ptr) ?? ""; }
            finally { Native.LicenseOS_Free(ptr); }
        }

        /// <summary>
        /// Report a usage metric to the server (usage-based licensing).
        /// </summary>
        public void ReportUsage(string metricName, long value)
        {
            ThrowIfDisposed();
            if (Native.LicenseOS_ReportUsage(_handle, metricName, value) == -1)
                throw new LicenseOSException(GetLastError() ?? "Failed to report usage");
        }

        // ── IDisposable ───────────────────────────────────────────────────────

        public void Dispose()
        {
            if (_disposed) return;
            _disposed = true;
            Native.LicenseOS_Destroy(_handle);
            _handle = -1;
        }

        // ── Private ───────────────────────────────────────────────────────────

        private string? GetLastError()
        {
            var ptr = Native.LicenseOS_GetLastError(_handle);
            if (ptr == IntPtr.Zero) return null;
            try   { return Marshal.PtrToStringUTF8(ptr); }
            finally { Native.LicenseOS_Free(ptr); }
        }

        private void ThrowIfDisposed()
        {
            if (_disposed) throw new ObjectDisposedException(nameof(Client));
        }
    }
}
