# LicenseOS SDK

Tích hợp license validation vào ứng dụng của bạn.

Hỗ trợ: **Go** · **C++** · **C#/.NET** · **Python**

---

## Mục lục

1. [Go SDK](#1-go-sdk)
2. [C / C++ (native library)](#2-c--c-native-library)
3. [C# / .NET (P/Invoke)](#3-c-net-pinvoke)
4. [Python](#4-python)
5. [Offline Mode](#5-offline-mode)
6. [Cấu hình nâng cao](#6-cấu-hình-nâng-cao)

---

## 1. Go SDK

### Cài đặt

```bash
go get github.com/anhtu97/license-system-sdk
```

### Cách dùng cơ bản

```go
package main

import (
    "log"
    licenseos "github.com/anhtu97/license-system-sdk"
)

func main() {
    client, err := licenseos.New(licenseos.Config{
        ServerURL:  "https://license.myapp.com",
        LicenseKey: "LIC-XXXX-XXXX-XXXX-XXXX",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 1. Validate license (kèm fallback offline)
    info, err := client.Validate()
    if err != nil {
        log.Fatal("License không hợp lệ:", err)
    }
    log.Printf("Status: %s | Plan: %s", info.Status, info.PlanName)

    // 2. Đăng ký máy lần đầu
    _ = client.Activate()

    // 3. Mở session (kiểm tra seat limit)
    if _, err := client.StartSession(); err != nil {
        log.Fatal("Không thể mở session:", err)
    }
    defer client.StopSession()

    // 4. Kiểm tra feature
    if client.HasFeature("export_pdf") {
        // bật tính năng export PDF
    }

    // 5. Báo cáo usage (cho gói usage-based)
    _ = client.ReportUsage("api_calls", 1)
}
```

### API Reference

| Method | Mô tả |
|---|---|
| `New(Config) (*Client, error)` | Khởi tạo client |
| `Validate() (*LicenseInfo, error)` | Kiểm tra license, fallback offline khi mất mạng |
| `Activate() error` | Đăng ký máy với server (idempotent) |
| `StartSession() (*Session, error)` | Mở session, bắt đầu heartbeat tự động |
| `StopSession() error` | Đóng session, dừng heartbeat |
| `HasFeature(name string) bool` | Kiểm tra feature flag |
| `FeatureValue(name string) interface{}` | Lấy giá trị raw của feature |
| `GetFingerprint() string` | Hardware fingerprint của máy (SHA-256 hex) |
| `ReportUsage(metric string, value int64) error` | Gửi usage metric |

### Config

```go
licenseos.Config{
    ServerURL:        "https://license.myapp.com",  // bắt buộc
    LicenseKey:       "LIC-XXXX-XXXX-XXXX-XXXX",   // bắt buộc
    CACertPath:       "./certs/ca.crt",             // optional: pin CA cert
    CertPath:         "./certs/client.crt",         // optional: mTLS client cert
    KeyPath:          "./certs/client.key",         // optional: mTLS client key
    CachePath:        "./licenseos.cache",           // optional: cache offline token
    OfflinePublicKey: "-----BEGIN PUBLIC KEY-----\n...", // optional: offline mode
}
```

---

## 2. C / C++ (native library)

### Build

```bash
cd sdk/clib

# Linux (cần gcc):
make linux
# → dist/linux/liblicenseos.so + licenseos.h

# Linux không có toolchain (dùng Docker):
make linux-docker

# macOS:
make macos
# → dist/macos/liblicenseos.dylib + licenseos.h

# macOS Apple Silicon:
make macos-arm64

# Windows (cần mingw-w64):
# sudo apt install gcc-mingw-w64-x86-64
make windows
# → dist/windows/licenseos.dll + licenseos.h
```

### Include

Copy `licenseos.h` vào project, link với thư viện tương ứng:

```cmake
# CMakeLists.txt
target_include_directories(myapp PRIVATE path/to/licenseos)
target_link_libraries(myapp PRIVATE ${CMAKE_SOURCE_DIR}/libs/liblicenseos.so)
```

### Ví dụ C++

```cpp
#include "licenseos.h"
#include <iostream>

int main() {
    // Khởi tạo
    licenseos_handle_t h = LicenseOS_Create(
        "https://license.myapp.com",
        "LIC-XXXX-XXXX-XXXX-XXXX",
        "",  // ca_cert_path
        "",  // cert_path (mTLS)
        "",  // key_path  (mTLS)
        ""   // offline_public_key
    );
    if (h == LICENSEOS_ERROR) return 1;

    // Validate
    char* info = LicenseOS_Validate(h);
    if (!info) {
        char* err = LicenseOS_GetLastError(h);
        std::cerr << "Error: " << err << '\n';
        LicenseOS_Free(err);
        LicenseOS_Destroy(h);
        return 1;
    }
    std::cout << info << '\n';
    LicenseOS_Free(info);  // bắt buộc free mọi string trả về

    LicenseOS_Activate(h);
    LicenseOS_StartSession(h);

    if (LicenseOS_HasFeature(h, "export_pdf"))
        std::cout << "PDF export enabled\n";

    // ... app logic ...

    LicenseOS_StopSession(h);
    LicenseOS_Destroy(h);
    return 0;
}
```

### Lưu ý quan trọng — Memory

Mọi hàm trả về `char*` đều cấp phát bộ nhớ mới. **Caller phải gọi `LicenseOS_Free(ptr)` khi xong.**

```cpp
char* fp = LicenseOS_GetFingerprint(h);
// ... dùng fp ...
LicenseOS_Free(fp);  // không được bỏ qua
```

### API C

| Hàm | Mô tả |
|---|---|
| `LicenseOS_Create(...)` | Khởi tạo, trả về handle |
| `LicenseOS_Destroy(handle)` | Giải phóng, stop session |
| `LicenseOS_Validate(handle)` | Trả về JSON string hoặc NULL |
| `LicenseOS_Activate(handle)` | Đăng ký máy, trả về 0/-1 |
| `LicenseOS_StartSession(handle)` | Mở session, trả về 0/-1 |
| `LicenseOS_StopSession(handle)` | Đóng session, trả về 0/-1 |
| `LicenseOS_HasFeature(handle, name)` | Trả về 1/0 |
| `LicenseOS_FeatureValue(handle, name)` | Trả về JSON string hoặc NULL |
| `LicenseOS_GetFingerprint(handle)` | Trả về hex string |
| `LicenseOS_GetLastError(handle)` | Trả về error string hoặc NULL |
| `LicenseOS_ReportUsage(handle, metric, value)` | Gửi metric, trả về 0/-1 |
| `LicenseOS_Free(ptr)` | Free string trả về từ SDK |

---

## 3. C# / .NET (P/Invoke)

### Cài đặt

1. Copy `sdk/clib/dotnet/LicenseOS.cs` vào project
2. Copy native library cạnh executable:
   - Windows: `licenseos.dll`
   - Linux: `liblicenseos.so`
   - macOS: `liblicenseos.dylib`

### Ví dụ

```csharp
using LicenseOS;

using var client = new Client(
    serverUrl:  "https://license.myapp.com",
    licenseKey: "LIC-XXXX-XXXX-XXXX-XXXX"
);

// Validate
LicenseInfo info = client.Validate(); // throws LicenseOSException on failure
Console.WriteLine($"Status: {info.Status} | Plan: {info.PlanName}");

// Activate + session
client.Activate();
client.StartSession();

if (client.HasFeature("api_access"))
    EnableApiModule();

// Report usage
client.ReportUsage("api_calls", 1);

// Dispose() tự gọi StopSession() + Destroy() khi ra khỏi using block
```

### NuGet (tự host — tuỳ chọn)

Nếu muốn đóng gói thành NuGet package, thêm `.csproj`:

```xml
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net6.0</TargetFramework>
    <PackageId>LicenseOS.SDK</PackageId>
    <Version>1.0.0</Version>
  </PropertyGroup>
  <ItemGroup>
    <None Include="runtimes/win-x64/native/licenseos.dll" Pack="true"
          PackagePath="runtimes/win-x64/native/" />
    <None Include="runtimes/linux-x64/native/liblicenseos.so" Pack="true"
          PackagePath="runtimes/linux-x64/native/" />
    <None Include="runtimes/osx-x64/native/liblicenseos.dylib" Pack="true"
          PackagePath="runtimes/osx-x64/native/" />
  </ItemGroup>
</Project>
```

---

## 4. Python

Dùng `ctypes` để gọi native library:

```python
import ctypes, json, sys

# Load library
if sys.platform == "win32":
    lib = ctypes.CDLL("licenseos.dll")
elif sys.platform == "darwin":
    lib = ctypes.CDLL("liblicenseos.dylib")
else:
    lib = ctypes.CDLL("./liblicenseos.so")

# Setup return types
lib.LicenseOS_Create.restype = ctypes.c_int
lib.LicenseOS_Validate.restype = ctypes.c_char_p
lib.LicenseOS_GetLastError.restype = ctypes.c_char_p
lib.LicenseOS_GetFingerprint.restype = ctypes.c_char_p
lib.LicenseOS_HasFeature.restype = ctypes.c_int

# Create client
handle = lib.LicenseOS_Create(
    b"https://license.myapp.com",
    b"LIC-XXXX-XXXX-XXXX-XXXX",
    b"", b"", b"", b""
)

# Validate
raw = lib.LicenseOS_Validate(handle)
if raw is None:
    err = lib.LicenseOS_GetLastError(handle)
    raise RuntimeError(f"License invalid: {err.decode()}")

info = json.loads(raw)
print(f"Status: {info['status']} | Plan: {info['plan_name']}")

lib.LicenseOS_Free(raw)
lib.LicenseOS_Activate(handle)
lib.LicenseOS_StartSession(handle)

# Check features
if lib.LicenseOS_HasFeature(handle, b"export_pdf"):
    print("PDF export: enabled")

# Cleanup
lib.LicenseOS_StopSession(handle)
lib.LicenseOS_Destroy(handle)
```

---

## 5. Offline Mode

Khi app mất kết nối internet, SDK tự động fallback sang **offline token** đã cache từ lần validate online trước.

Offline token được ký bằng **ECDSA P-256** — không thể forge mà không có private key trên server.

### Setup

**Bước 1** — Lấy public key từ server:
```bash
curl https://license.myapp.com/v1/offline-pubkey
# → trả về PEM block
```

**Bước 2** — Nhúng vào config:

```go
// Go
client, _ := licenseos.New(licenseos.Config{
    ServerURL:        "https://license.myapp.com",
    LicenseKey:       "LIC-XXXX-XXXX-XXXX-XXXX",
    OfflinePublicKey: `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE...
-----END PUBLIC KEY-----`,
})
```

```cpp
// C++
LicenseOS_Create(url, key, "", "", "", offlinePubKeyPem);
```

```csharp
// C#
new Client(serverUrl, licenseKey, offlinePublicKey: offlinePubKeyPem);
```

### Thời gian offline tối đa

Cấu hình trong Admin Dashboard → Products → Plan → **Offline Grace Days**.

---

## 6. Cấu hình nâng cao

### mTLS (Mutual TLS)

Khi server bật `REQUIRE_MTLS=true`, client phải có certificate:

```go
client, _ := licenseos.New(licenseos.Config{
    ServerURL:  "https://license.myapp.com",
    LicenseKey: "LIC-XXXX-XXXX-XXXX-XXXX",
    CACertPath: "./certs/ca.crt",
    CertPath:   "./certs/client.crt",
    KeyPath:    "./certs/client.key",
})
```

Lấy client cert từ Admin → Licenses → [License] → **Issue Certificate**.

### Custom Logger (Go only)

```go
type MyLogger struct{}
func (l *MyLogger) Debugf(f string, a ...interface{}) { /* ... */ }
func (l *MyLogger) Infof(f string, a ...interface{})  { /* ... */ }
func (l *MyLogger) Errorf(f string, a ...interface{}) { /* ... */ }

client, _ := licenseos.New(licenseos.Config{
    Logger: &MyLogger{},
    // ...
})
```

### Lấy CA certificate

```bash
curl https://license.myapp.com/ca.crt -o licenseos-ca.crt
```

---

*LicenseOS SDK — Go · C++ · C# · Python*
