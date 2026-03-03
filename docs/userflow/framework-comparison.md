# Framework Comparison

Comprehensive comparison of all testing framework adapters in the `pkg/userflow` package. Use this guide to select the right adapter for your testing requirements.

## Adapter Summary

| Adapter | Interface | Platform | Tool | Source File |
|---------|-----------|----------|------|-------------|
| PlaywrightCLIAdapter | BrowserAdapter | Web | Playwright + CDP | `playwright_cli_adapter.go` |
| SeleniumAdapter | BrowserAdapter | Web | Selenium WebDriver | `selenium_adapter.go` |
| CypressCLIAdapter | BrowserAdapter | Web | Cypress CLI | `cypress_adapter.go` |
| PuppeteerAdapter | BrowserAdapter | Web | Puppeteer + Node.js | `puppeteer_adapter.go` |
| ADBCLIAdapter | MobileAdapter | Android | ADB | `adb_cli_adapter.go` |
| AppiumAdapter | MobileAdapter | Android, iOS | Appium 2.0 | `appium_adapter.go` |
| MaestroCLIAdapter | MobileAdapter | Android, iOS | Maestro CLI | `maestro_adapter.go` |
| EspressoAdapter | MobileAdapter | Android | Gradle + ADB | `espresso_adapter.go` |
| RobolectricAdapter | BuildAdapter | Android (JVM) | Gradle | `robolectric_adapter.go` |
| GRPCCLIAdapter | GRPCAdapter | API | grpcurl | `adapter_grpc.go` |
| GorillaWebSocketAdapter | WebSocketFlowAdapter | API | gorilla/websocket | `adapter_websocket_flow.go` |
| HTTPAPIAdapter | APIAdapter | API | pkg/httpclient | `http_api_adapter.go` |
| TauriCLIAdapter | DesktopAdapter | Desktop | Tauri WebDriver | `tauri_cli_adapter.go` |
| GoCLIAdapter | BuildAdapter | Build | go build/test | `go_cli_adapter.go` |
| GradleCLIAdapter | BuildAdapter | Build | Gradle | `gradle_cli_adapter.go` |
| NPMCLIAdapter | BuildAdapter | Build | npm/npx | `npm_cli_adapter.go` |
| CargoCLIAdapter | BuildAdapter | Build | Cargo | `cargo_cli_adapter.go` |
| ProcessCLIAdapter | ProcessAdapter | System | subprocess | `process_cli_adapter.go` |

## Feature Matrix: Browser Adapters

| Feature | Playwright | Selenium | Cypress | Puppeteer |
|---------|------------|----------|---------|-----------|
| **Chrome** | Yes | Yes | Yes | Yes |
| **Firefox** | Yes | Yes | Yes | Experimental |
| **Safari/WebKit** | Yes (WebKit) | Yes | No | No |
| **Edge** | Yes (Chromium) | Yes | Yes | No |
| **Headless mode** | Yes | Yes | Yes | Yes |
| **Network intercept** | Via CDP (not CLI) | No (use proxy) | No (CLI) | No (CLI) |
| **Screenshots** | Yes | Yes (base64) | Yes (file) | Yes (base64) |
| **JavaScript eval** | Yes | Yes | Yes (base64) | Yes (base64) |
| **Session model** | CDP WebSocket | HTTP session | Per-spec process | WS endpoint |
| **Speed** | Fast | Medium | Slow | Medium |
| **Container exec** | podman exec | HTTP server | npx (local) | podman exec fallback |
| **Setup complexity** | Low | Medium (Grid) | Medium (npm) | Low |
| **State persistence** | Per-CDP connection | Per-session | None | Per-browser |

## Feature Matrix: Mobile Adapters

| Feature | ADB CLI | Appium | Maestro | Espresso |
|---------|---------|--------|---------|----------|
| **Android** | Yes | Yes | Yes | Yes |
| **iOS** | No | Yes | Yes | No |
| **Device required** | Yes | Yes | Yes | Yes |
| **Emulator support** | Yes | Yes | Yes | Yes |
| **No device (JVM)** | No | No | No | No |
| **Instrumented tests** | No | Via shell | No | Yes (native) |
| **Element selectors** | No | W3C WebDriver | Text/accessibility | Espresso matchers |
| **Screenshots** | Yes (exec-out) | Yes (base64) | Yes (file) | Yes (exec-out) |
| **Cross-platform** | Android only | Android + iOS | Android + iOS | Android only |
| **Setup complexity** | Low (ADB only) | High (server) | Low (CLI) | Medium (Gradle) |
| **Protocol** | ADB shell | W3C WebDriver | CLI/YAML | ADB + Gradle |
| **Speed** | Fast | Medium | Medium | Medium |

## Feature Matrix: API and Protocol Adapters

| Feature | HTTP API | gRPC CLI | WebSocket |
|---------|----------|----------|-----------|
| **Protocol** | HTTP/HTTPS | gRPC (HTTP/2) | WebSocket |
| **Request type** | REST JSON | Protobuf (JSON wire) | Binary/Text frames |
| **Streaming** | SSE (read only) | Server streaming | Bidirectional |
| **Auth support** | JWT, API key | Headers, TLS certs | Headers on connect |
| **Connection** | Per-request | Per-command | Persistent |
| **Thread-safe** | Yes (http.Client) | N/A (CLI) | Yes (mutexes) |
| **Health check** | HTTP status | grpc.health.v1 | Connection check |
| **Tool required** | None (stdlib) | grpcurl | gorilla/websocket |

## Architecture Diagrams

### Browser Adapter Architecture

```
                    +-------------------+
                    |  BrowserAdapter   |
                    |    (interface)    |
                    +--------+----------+
                             |
          +----------+-------+-------+-----------+
          |          |               |            |
  +-------+--+ +----+-----+ +------+----+ +-----+-------+
  |Playwright | | Selenium | |  Cypress  | | Puppeteer   |
  |CLI Adapter| | Adapter  | |CLI Adapter| | Adapter     |
  +-----+-----+ +----+----+ +-----+-----+ +------+------+
        |             |            |              |
  podman exec    HTTP/JSON    npx cypress      node <script>
  node -e        /session     run --spec       -> puppeteer
        |             |            |              |
     CDP WS      WebDriver     Cypress         CDP WS
        |          Server       Runner            |
     Browser      Browser      Browser         Browser
```

### Mobile Adapter Architecture

```
                    +-------------------+
                    |  MobileAdapter    |
                    |    (interface)    |
                    +--------+----------+
                             |
          +----------+-------+-------+-----------+
          |          |               |            |
  +-------+--+ +----+-----+ +------+----+ +-----+-------+
  |  ADB CLI | |  Appium  | |  Maestro  | |  Espresso   |
  |  Adapter | | Adapter  | |CLI Adapter| |  Adapter    |
  +-----+----+ +----+-----+ +-----+----+ +------+------+
        |             |            |         |        |
    adb shell    HTTP/JSON    maestro     adb     gradlew
        |             |        test       shell   connected
     Device      Appium         |           |    AndroidTest
                 Server      Device      Device     |
                    |                            Device
               UiAutomator2
               or XCUITest
```

### Protocol Adapter Architecture

```
  +------------------+    +------------------+    +------------------+
  |   HTTPAPIAdapter |    |  GRPCCLIAdapter  |    |GorillaWebSocket  |
  |   (APIAdapter)   |    |  (GRPCAdapter)   |    |  Adapter         |
  +--------+---------+    +--------+---------+    +--------+---------+
           |                       |                       |
    net/http.Client          grpcurl CLI            gorilla/websocket
           |                       |                       |
    HTTP/HTTPS REST         gRPC (HTTP/2)          WebSocket (ws/wss)
           |                       |                       |
       API Server            gRPC Server           WS Server
```

## When to Use Each Framework

### Browser Testing

| Scenario | Recommended Adapter |
|----------|--------------------|
| General web testing with container infrastructure | **Playwright** |
| Multi-browser testing (including Safari) | **Selenium** |
| Project already uses Cypress | **Cypress** |
| Simple scripting with Node.js | **Puppeteer** |
| CI/CD with minimal dependencies | **Selenium** (standalone Docker) |

### Mobile Testing

| Scenario | Recommended Adapter |
|----------|--------------------|
| Android-only, fast device interaction | **ADB CLI** |
| Cross-platform (Android + iOS) | **Appium** |
| Quick UI flow verification | **Maestro** |
| Instrumented test execution | **Espresso** |
| JVM unit tests, no device | **Robolectric** |
| Full E2E with element queries | **Appium** |

### Protocol Testing

| Scenario | Recommended Adapter |
|----------|--------------------|
| REST API testing | **HTTP API** |
| gRPC service testing | **gRPC CLI** |
| Real-time WebSocket flows | **WebSocket** |
| SSE event streams | **HTTP API** |

## Cost and Licensing

| Framework | License | Cost |
|-----------|---------|------|
| Playwright | Apache 2.0 | Free |
| Selenium | Apache 2.0 | Free |
| Cypress | MIT | Free (open source), paid dashboard |
| Puppeteer | Apache 2.0 | Free |
| Appium | Apache 2.0 | Free |
| Maestro | Apache 2.0 | Free (CLI), paid cloud |
| Espresso | Apache 2.0 | Free (part of AndroidX) |
| Robolectric | MIT | Free |
| grpcurl | MIT | Free |
| gorilla/websocket | BSD-2-Clause | Free |

All adapters in this package use open-source tools with permissive licenses. No adapter requires a paid service for local execution.

## CI/CD Considerations

| Adapter | Docker Available | Emulator Needed | Typical CI Time |
|---------|-----------------|-----------------|-----------------|
| Playwright | Yes (mcr.microsoft.com/playwright) | No | Fast (~seconds/test) |
| Selenium | Yes (selenium/standalone-chrome) | No | Medium |
| Cypress | Yes (cypress/included) | No | Slow (~seconds/spec) |
| Puppeteer | Yes (custom Node.js) | No | Medium |
| Appium | Yes (appium/appium) | Yes (Android) | Slow |
| Maestro | Partial | Yes | Medium |
| Espresso | Partial | Yes | Slow |
| Robolectric | Yes (any JDK image) | No | Fast |
| gRPC CLI | Yes (fullstorydev/grpcurl) | No | Fast |
| WebSocket | Yes (any Go image) | No | Fast |

## Documentation Index

| Adapter | Documentation |
|---------|--------------|
| Playwright | [browser-adapter.md](browser-adapter.md) |
| Selenium | [selenium-adapter.md](selenium-adapter.md) |
| Cypress | [cypress-adapter.md](cypress-adapter.md) |
| Puppeteer | [puppeteer-adapter.md](puppeteer-adapter.md) |
| ADB CLI | [mobile-adapter.md](mobile-adapter.md) |
| Appium | [appium-adapter.md](appium-adapter.md) |
| Maestro | [maestro-adapter.md](maestro-adapter.md) |
| Espresso | [espresso-adapter.md](espresso-adapter.md) |
| Robolectric | [robolectric-adapter.md](robolectric-adapter.md) |
| gRPC CLI | [grpc-adapter.md](grpc-adapter.md) |
| WebSocket | [websocket-adapter.md](websocket-adapter.md) |
| HTTP API | [api-adapter.md](api-adapter.md) |
| Desktop (Tauri) | [desktop-adapter.md](desktop-adapter.md) |
| Build (Go/npm/Gradle/Cargo) | [build-adapter.md](build-adapter.md) |
| Process | [process-adapter.md](process-adapter.md) |
