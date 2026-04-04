# PicoClaw 資安審計報告 (2026-03)

## 背景

針對 Raspberry Pi 5 上運行中的 PicoClaw AI 代理服務進行資安掃描，評估外網攻擊面、API 金鑰洩漏風險及潛在入侵向量。審計日期：2026-03-14。

---

## 現況確認（審計時實際運行狀態）

| 項目 | 狀態 |
|------|------|
| 運行程序 | `/usr/local/bin/picoclaw gateway` (PID 1205) |
| 執行身份 | `pi` 使用者（非 root） |
| 監聽位址 | `127.0.0.1:18790`（僅本機） |
| 對外暴露端口 | **無**（launcher UI port 18800 未啟動） |

**整體暴露面評估：極小。** 外部攻擊者無法直接存取此服務。

---

## 安全發現（依嚴重程度排序）

### 🔴 高風險

#### H1: Launcher UI 完全無認證

- **位置**：`cmd/picoclaw-launcher/internal/server/server.go`
- **問題**：`/api/config`（讀寫設定）、`/api/auth/login`、`/api/auth/logout` 等端點完全無身份驗證
- **目前狀態**：Launcher 目前未運行，無立即威脅
- **風險**：若啟動 launcher（尤其加 `-public` flag），任何人可讀取/修改設定、觸發 OAuth 登入、竊取 API 金鑰
- **建議**：啟動 launcher 前務必確保只綁定 `127.0.0.1`，絕不使用 `-public` flag
- **修復狀態**：✅ 已加入啟動時安全警告（見下方實施記錄 H2）

#### H2: `-public` flag 可將 Launcher UI 暴露到外網

- **位置**：`cmd/picoclaw-launcher/main.go`，`DefaultPort = 18800`
- **問題**：加上 `-public` 參數會改綁定 `0.0.0.0:18800`，整個設定介面含 API 金鑰管理都暴露到外網，且無認證
- **建議**：此 flag 等同「開門揖盜」，正式部署絕不使用
- **修復狀態**：✅ 已在 `-public` 啟動時顯示醒目安全警告

---

### 🟠 中風險

#### M1: Shell 執行防護可被設定關閉

- **位置**：`pkg/tools/shell.go`
- **問題**：`tools.exec.enable_deny_patterns: false` 可停用所有指令黑名單，允許任意 shell 指令
- **現況**：預設啟用（安全）；`shell.go:121` 已有 `fmt.Println` 警告
- **修復狀態**：✅ 已在 gateway 啟動時額外顯示 `⚠ Security` 警告

#### M2: Pico WebSocket 頻道預設允許任意來源 CORS

- **位置**：`pkg/channels/pico/pico.go` 第 69-81 行
- **問題**：若 `allow_origins` 未設定，WebSocket 握手接受所有 Origin
- **建議**：設定中明確指定 `allow_origins`，不留空
- **修復狀態**：✅ 已在頻道建立時透過 `logger.WarnC` 記錄警告

#### M3: OAuth Token 以明文存放於磁碟

- **位置**：`~/.picoclaw/auth.json`（權限 0o600）
- **問題**：Anthropic/Google/OpenAI 的 OAuth token 以明文 JSON 存放
- **緩解**：檔案權限 0o600（僅 pi 使用者可讀）
- **修復狀態**：⏸ 可接受現況；若需更高安全性可考慮加密存放（長期項目）

#### M4: 主設定檔 config.json 權限

- **位置**：`~/.picoclaw/config.json`
- **審計時描述**：疑慮為 0o644（世界可讀）
- **實際確認**：`-rw------- (0o600)`，`SaveConfig()` 已正確使用 `fileutil.WriteFileAtomic(path, data, 0o600)`
- **修復狀態**：✅ 無需修改，已符合安全標準

#### M5: Workspace 限制可被環境變數停用

- **位置**：`pkg/config/config.go`
- **問題**：`PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=false` 可讓 AI 代理存取整個檔案系統
- **建議**：確保此環境變數未被設定為 false
- **修復狀態**：✅ 已在 gateway 啟動時顯示 `⚠ Security` 警告

---

### 🟡 低風險

#### L1: Pico 頻道允許 token 放在 URL query 參數

- **位置**：`pkg/channels/pico/pico.go`
- **問題**：`allow_token_query: true` 時，token 出現在 URL 中，可能被 proxy/系統日誌記錄
- **建議**：使用預設的 Bearer token（Authorization header），不啟用 query 參數模式
- **修復狀態**：⏸ 文件警告已足夠；預設為 false（安全）

#### L2: Launcher API 無速率限制

- **位置**：`cmd/picoclaw-launcher/internal/server/server.go`
- **問題**：若 Launcher 運行，無防暴力攻擊或 DoS 保護
- **建議**：如需對外暴露，加上 nginx reverse proxy 並設定 rate limiting
- **修復狀態**：⏸ 長期項目

#### L3: channel `allow_from` 預設為空（不限制使用者）

- **位置**：`pkg/channels/base.go`
- **問題**：未設定 `allow_from` 時，任何人（例如知道 bot 帳號的人）可傳訊息給 bot
- **建議**：在每個頻道設定中明確列出允許的使用者 ID
- **修復狀態**：✅ 已在 gateway 啟動時對所有已啟用但 `allow_from` 為空的頻道顯示 `⚠ Security` 警告

#### L4: 硬編碼 Google/OpenAI OAuth Client ID 於原始碼

- **位置**：`pkg/auth/oauth.go` 第 47-50 行
- **說明**：這是 Google Cloud Code Assist 和 OpenAI 的公開 OAuth client credential，本身不構成安全問題，屬已知設計
- **修復狀態**：✅ 無需修改

---

## 實施記錄

### 2026-03-14 完成的修改

#### 1. Launcher `-public` 安全警告

**檔案**：`cmd/picoclaw-launcher/main.go`

使用 `-public` 旗標啟動時，在啟動 banner 中顯示醒目的安全警告：

```
⚠ SECURITY WARNING: -public flag is active.
The configuration interface (including API keys) is
accessible from ALL network interfaces WITHOUT authentication.
Only use -public on trusted networks. Never expose to the internet.
```

#### 2. Pico WebSocket CORS 警告

**檔案**：`pkg/channels/pico/pico.go`

在 `NewPicoChannel()` 中，當 `allow_origins` 未設定時，透過 `logger.WarnC` 記錄：

```
allow_origins is not configured: WebSocket connections will be accepted
from any Origin. Set allow_origins in config to restrict access.
```

#### 3. Gateway 啟動安全檢查

**檔案**：`cmd/picoclaw/internal/gateway/helpers.go`

新增 `printSecurityWarnings(cfg)` 函數，於每次 `picoclaw gateway` 啟動時執行以下檢查：

| 檢查項目 | 觸發條件 | 警告訊息 |
|---------|---------|---------|
| M1 | `exec.enable_deny_patterns: false` | shell 安全守衛被關閉 |
| M5 | `restrict_to_workspace: false` | AI 代理可存取整個檔案系統 |
| L3 | 已啟用頻道的 `allow_from` 為空 | 任何人可傳訊息給 bot |

涵蓋所有 14 個頻道：telegram、discord、slack、feishu、dingtalk、qq、wecom、wecom_app、wecom_aibot、line、onebot、pico、whatsapp、maixcam。

---

## 主要攻擊向量（供參考）

1. 若有人能本機登入（或 SSH 進入）→ 可存取 `~/.picoclaw/config.json` 獲取 API 金鑰
2. 若啟動 launcher 且使用 `-public` flag → 遠端攻擊者可完全控制設定
3. 若 AI 代理被操控（prompt injection）→ 可嘗試執行 shell 指令（有黑名單防護，但不完全）

---

## 剩餘行動項目

### 短期（如需啟動 Launcher）

- [ ] 只在本機使用 Launcher（不加 `-public` flag）
- [ ] 或在 Launcher 前加 nginx 並設 HTTP Basic Auth 或 IP 白名單

### 長期（如需公開部署）

- [ ] 為 Launcher API 加入認證機制
- [ ] 為 Pico WebSocket 設定 `allow_origins`
- [ ] 考慮加 VPN 或 Cloudflare Tunnel 而非直接公開
- [ ] Launcher API 加入速率限制（防 DoS）

---

## 關鍵檔案索引

| 檔案 | 說明 |
|------|------|
| `pkg/tools/shell.go` | Shell 執行與黑名單 |
| `pkg/auth/oauth.go` | OAuth 實作與 hardcoded client ID |
| `pkg/auth/store.go` | 憑證存放（0o600） |
| `pkg/channels/manager.go` | HTTP server 與 rate limiting |
| `cmd/picoclaw-launcher/main.go` | Launcher 入口（`-public` flag）|
| `cmd/picoclaw-launcher/internal/server/server.go` | 無認證的 API 端點 |
| `pkg/channels/pico/pico.go` | WebSocket CORS 設定 |
| `pkg/channels/base.go` | `allow_from` 執行邏輯 |
| `~/.picoclaw/config.json` | 實際運行設定（含 API 金鑰，0o600）|
| `~/.picoclaw/auth.json` | OAuth token 存放位置（0o600）|
