> 返回 [README](../../../README.zh.md)

# Line

PicoClaw 通过 LINE Messaging API 配合 Webhook 回调功能实现对 LINE 的支持。

## 配置

```json
{
  "channels": {
    "line": {
      "enabled": true,
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  }
}
```

| 字段                 | 类型   | 必填 | 描述                                       |
| -------------------- | ------ | ---- | ------------------------------------------ |
| enabled              | bool   | 是   | 是否启用 LINE Channel                      |
| channel_secret       | string | 是   | LINE Messaging API 的 Channel Secret       |
| channel_access_token | string | 是   | LINE Messaging API 的 Channel Access Token |
| webhook_path         | string | 否   | Webhook 的路径 (默认为 /webhook/line)      |
| allow_from           | array  | 否   | 用户ID白名单，空表示允许所有用户           |

## 设置流程

1. 前往 [LINE Developers Console](https://developers.line.biz/console/) 创建一个服务提供商和一个 Messaging API Channel
2. 获取 Channel Secret 和 Channel Access Token
3. 配置Webhook:
   - LINE 要求 Webhook 必须使用 HTTPS 协议，因此需要部署一个支持 HTTPS 的服务器，或者使用反向代理工具如 ngrok 将本地服务器暴露到公网
   - PicoClaw 现在使用共享的 Gateway HTTP 服务器来接收所有渠道的 webhook 回调，默认监听地址为 127.0.0.1:18790
   - 将 Webhook URL 设置为 `https://your-domain.com/webhook/line`，然后将外部域名反向代理到本机的 Gateway（默认端口 18790）
   - 启用 Webhook 并验证 URL
4. 将 Channel Secret 和 Channel Access Token 填入配置文件中

## 加入群組

PicoClaw 支援 LINE 群組（Group）與聊天室（Room）。將機器人加入群組前，需先在 LINE Developers Console 開啟群組聊天權限：

1. 進入 Messaging API 設定頁面
2. 找到 **「Allow bot to join group chats」** 並開啟

完成後即可在群組內邀請機器人加入。

## 圖片（Vision）支援

使用 `claude-cli` provider 時，用戶傳送的圖片會自動啟用視覺理解模式：

- LINE channel 下載圖片後存入 media store（`media://` 引用）
- Agent loop 解析為 `data:image/jpeg;base64,...` 格式
- claude-cli provider 偵測到圖片後，改用 `--input-format stream-json --output-format stream-json --verbose` 模式執行 claude CLI，將圖片以 Anthropic image content block 格式傳入
- 無圖片的純文字訊息維持原有路徑，不受影響

> **注意：** Vision 功能需要 claude CLI 版本支援 `--input-format stream-json`，且使用的 Claude 模型需具備視覺能力（如 claude-sonnet-4.x、claude-opus-4.x）。

### 群組中傳送圖片（Pending Image Buffer）

LINE 群組的限制：圖片與 @mention 文字是兩則獨立訊息，無法同時傳送。PicoClaw 透過 **pending image buffer** 機制解決這個問題：

**使用流程：**
1. 在群組中先傳送圖片（不需要 @bot）
2. 在 **60 秒內**發送 `@bot 你的問題`
3. Bot 自動將步驟 1 的圖片附上一起送給 Claude

**行為細節：**
- 同一張圖片在 60 秒 TTL 內可被多次 @mention 重複使用（每次都會帶著圖片）
- 超過 60 秒後 @mention，bot 會回覆「圖片已超過 60 秒等待時間，請重新傳送圖片後再 @我。」並中止本次請求
- 1 對 1 私聊不受此限制，直接傳圖即可

### 常見問題：無法重新邀請機器人

**現象：** 曾經邀請過機器人（但當時未啟用 Webhook），再次邀請時按鈕無法點選。

**原因：** LINE 已將機器人記錄為群組成員，即使它從未回應過。

**解決方法：**
1. 進入群組成員列表，找到機器人帳號
2. 將機器人**移除（踢出）**群組
3. 再重新邀請即可成功加入

### 常見問題：修改程式碼後功能沒有生效

**現象：** 程式碼已修改、`go build` 成功，但行為沒有改變。

**原因：** `go build ./...` 只在本地編譯，service 執行的是 `/usr/local/bin/picoclaw`（舊版本）。

**解決方法：**
```bash
sudo systemctl stop picoclaw
go build -o /tmp/picoclaw-new ./cmd/picoclaw/
sudo cp /tmp/picoclaw-new /usr/local/bin/picoclaw
sudo systemctl start picoclaw
```
