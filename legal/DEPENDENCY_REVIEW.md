# Windows Distribution Dependency Review

Review date: 2026-09-12. Target: Windows amd64, CGO disabled.

## Outcome

The reviewed Go module license texts permit commercial use subject to their conditions; no noncommercial-only restriction was found in those module license files. This is a technical inventory and notice review, not a legal opinion or a guarantee of compliance in every jurisdiction. Microsoft software remains governed by Microsoft's own terms.

The practical changes are:

- Preserve the upstream license texts in `THIRD_PARTY_NOTICES.txt`, including nested notices, Go runtime/vendor notices, Unicode data terms and conservative Vite build-tool notices.
- Display the project's MIT license and third-party notices offline in the app's Developer Information dialog.
- Include `LICENSE` and `THIRD_PARTY_NOTICES.txt` alongside the executable in release ZIPs. The standalone executable also embeds the notice text through the frontend.
- Use Wails' `browser` WebView2 installation strategy. If the runtime is absent, the app offers to open Microsoft's official download page. It does not distribute or execute Microsoft's bootstrapper.
- Identify the app as independent of Microsoft and include the WebView2 SmartScreen disclosure and privacy links.

## Evidence and Scope

The previously published `2026.09.13` EXE was inspected using `go version -m`. Its SHA-256 was `87dff13b8713091a08c09fb06b693afe5de08e000996ebd9f7e7b8802abfd267`. It was built with Go `go1.27.1`, `CGO_ENABLED=0`, and `desktop,wv2runtime.embed,production`. Eighteen external Go modules were recorded. The original inventory is preserved in [released-2026.09.13-inventory.json](released-2026.09.13-inventory.json).

The current inventory is [dependency-inventory.json](dependency-inventory.json). It is regenerated from `go list -deps -json` with the Windows production build tags, rather than assuming that every module in `go.mod` ships in the EXE. `--binary` compares the build's actual module versions, Go toolchain and build settings against that graph and checks that the upstream bootstrapper payload is absent.

Each notice file has its original source path and SHA-256 recorded. Module-level notice collection is intentionally conservative: nested license files can describe source or build-time code that the linker does not include. Go's non-BoringCrypto build can still select stub packages from the `crypto/internal/boring` directory; preserving that directory's license does not imply that BoringSSL cryptography is linked.

## Go Modules in the Windows Dependency Graph

| Module | Version | Root license / additional material |
| --- | --- | --- |
| `git.sr.ht/~jackmordaunt/go-toast/v2` | `v2.0.3` | MIT OR Unlicense; MIT is sufficient for this distribution |
| `github.com/bep/debounce` | `v1.2.1` | MIT |
| `github.com/go-ole/go-ole` | `v1.3.0` | MIT |
| `github.com/google/uuid` | `v1.6.0` | BSD-3-Clause |
| `github.com/leaanthony/go-ansi-parser` | `v1.6.1` | MIT |
| `github.com/leaanthony/slicer` | `v1.6.0` | MIT |
| `github.com/leaanthony/u` | `v1.1.1` | MIT |
| `github.com/pkg/browser` | `v0.0.0-20240102092130-5ac0b6a4141c` | BSD-2-Clause |
| `github.com/pkg/errors` | `v0.9.1` | BSD-2-Clause |
| `github.com/rivo/uniseg` | `v0.4.7` | MIT; generated tables also reference Unicode terms |
| `github.com/samber/lo` | `v1.49.1` | MIT |
| `github.com/tkrajina/go-reflector` | `v0.5.8` | Apache-2.0 |
| `github.com/wailsapp/go-webview2` | `v1.0.22` | MIT; `webviewloader` has ISC terms |
| `github.com/wailsapp/mimetype` | `v1.4.1` | MIT |
| `github.com/wailsapp/wails/v2` | `v2.14.0` | MIT; nested MIT and Apache-2.0 notices preserved |
| `golang.org/x/net` | `v0.56.0` | BSD-3-Clause and PATENTS |
| `golang.org/x/sys` | `v0.46.0` | BSD-3-Clause and PATENTS |
| `golang.org/x/text` | `v0.39.0` | BSD-3-Clause and PATENTS |

MIT/ISC notices preserve copyright, permission and warranty text. BSD notices also preserve their redistribution conditions and applicable non-endorsement clauses. Apache-2.0 terms are included in full; no separate upstream NOTICE file was found in the selected module trees. Modified upstream files or future versions require a new review. Go's license and patent grant are also included.

## Frontend and Other Materials

The frontend uses authored HTML, CSS and JavaScript with Vite as a development dependency. No React, Vue, third-party font or icon package is imported in the frontend. Wails' injected JavaScript runtime is covered by its source notices. The Vite license file includes upstream bundled notices; it is included conservatively because generated frontend output may contain a modulepreload helper. This inventory does not claim to audit every npm build-time dependency as a separately redistributed runtime component.

Unicode data notices are obtained from [Unicode's official license](https://www.unicode.org/license.txt), with a local snapshot in [UNICODE-LICENSE.txt](UNICODE-LICENSE.txt). Source comments in uniseg's generated tables explicitly refer to Unicode's license. The runtime and dependency source packages can contain additional attribution material; the collection script is not a complete file-by-file license classifier or a proof of asset provenance.

## Microsoft Components

DISM, SFC, Defender, `shutdown.exe`, Explorer and Windows service APIs are called on the user's existing Windows installation. Those Windows executables and system DLLs are not copied into this application's package. The app itself supplies operation ordering, cancellation, UI, cache-reset logic and app updates.

The original release embedded a 1,793,816-byte WebView2 bootstrapper from the Wails module cache. Its SHA-256 was `7ebc4ce80143ef89cea86a61ea151502868db6caaa678b8b43660a66ace11c3a`. This was Microsoft executable redistribution, not merely a call to an installed Windows command.

The [official WebView2 distribution documentation](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/distribution) describes both packaging the bootstrapper and directing users to Microsoft's download page. The [official Evergreen terms endpoint](https://developer.microsoft.com/microsoft-edge/api/eula/webview2), used by Microsoft's download page, supplied the [local plain-text terms snapshot](MICROSOFT-WEBVIEW2-TERMS.txt). It contains distribution requirements including direct acquisition from Microsoft, downstream protective terms, notices and an indemnity provision. A generic third-party disclaimer does not satisfy those requirements by itself.

The selected remediation is to use the official download-page workflow and avoid bundling the bootstrapper. The separately installed runtime still has Microsoft terms and privacy behavior. The full Evergreen distribution terms are preserved for reference; they are not presented as a new license for this app or as proof that every clause applies identically to every installation scenario.

## Verification and Remaining Limits

- `scripts/generate_notices.py --check --binary <exe>` verifies module/toolchain/settings agreement, source notice checksums and absence of the upstream embedded bootstrapper payload.
- `scripts/build-release.ps1` regenerates notices before the final frontend build, then verifies the EXE and packages the notices. Use this script for release packaging. Ordinary `wails build` defaults can select a different WebView2 strategy.
- The app displays license text using `textContent`, so upstream text is not executed as HTML or JavaScript.
- The PowerShell packaging script must also be exercised on Windows. Cross-compilation and source review do not test native dialogs, a missing-WebView2 installation, or actual shutdown behavior on Windows.
- This review does not cover future dependency changes, third-party code provenance beyond the inspected sources, trademarks or patent clearance, jurisdiction-specific liability, or contractual terms for commercial support/services.
- The previous `2026.09.13` release lacks the new embedded notice viewer and contains the bootstrapper. The initial reviewed replacement was version `2026.09.14`. The current `2026.09.15` build retains that dependency inventory and browser installation strategy; its build and ZIP checksums are recorded in [build-verification.json](build-verification.json). Historical assets are not silently replaced.
