// LicenseOS C# example
//
// Add LicenseOS.cs to your project and place the native library next to the exe:
//   Windows : licenseos.dll
//   Linux   : liblicenseos.so
//   macOS   : liblicenseos.dylib

using LicenseOS;

// ── 1. Create client (using = auto-dispose on exit) ───────────────────────────
using var client = new Client(
    serverUrl:  "https://license.myapp.com",
    licenseKey: "LIC-XXXX-XXXX-XXXX-XXXX"
    // caCertPath:        "licenseos-ca.crt",     // optional: pin server cert
    // offlinePublicKey:  offlinePem               // optional: enable offline mode
);

// ── 2. Validate license ───────────────────────────────────────────────────────
LicenseInfo info;
try
{
    info = client.Validate();
}
catch (LicenseOSException ex)
{
    Console.Error.WriteLine($"License invalid: {ex.Message}");
    return 1;
}

Console.WriteLine($"License : {info.LicenseKey}");
Console.WriteLine($"Status  : {info.Status}");
Console.WriteLine($"Plan    : {info.PlanName}");
Console.WriteLine($"Expires : {info.ExpiresAt ?? "never"}");

if (info.Status != "active" && info.Status != "trial")
{
    Console.Error.WriteLine("License is not active.");
    return 1;
}

// ── 3. Register machine ───────────────────────────────────────────────────────
try   { client.Activate(); }
catch { /* non-fatal if already activated */ }

// ── 4. Open session ───────────────────────────────────────────────────────────
try
{
    client.StartSession();
    Console.WriteLine("Session started (heartbeat running).");
}
catch (LicenseOSException ex)
{
    Console.Error.WriteLine($"Cannot start session: {ex.Message}");
    return 1;
}

// ── 5. Gate features ──────────────────────────────────────────────────────────
if (client.HasFeature("export_pdf"))
    Console.WriteLine("PDF export  : ENABLED");

if (client.HasFeature("api_access"))
    Console.WriteLine("API access  : ENABLED");

string? maxUsers = client.GetFeatureValue("max_users");
if (maxUsers != null)
    Console.WriteLine($"Max users   : {maxUsers}");

Console.WriteLine($"Fingerprint : {client.GetFingerprint()}");

// ── 6. Your application ───────────────────────────────────────────────────────
Console.WriteLine("\nApplication running. Press Enter to exit.");
Console.ReadLine();

// ── 7. Cleanup (also called automatically by Dispose) ─────────────────────────
client.StopSession();
// Dispose() is called by the `using` statement → LicenseOS_Destroy() → session released

return 0;
