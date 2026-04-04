---
name: knowledge-review
description: "從 Obsidian vault 隨機挑選一篇筆記改寫成科普短文。Trigger: 知識複習、科普文、knowledge review、每日知識、請執行知識複習。"
---

# Knowledge Review — 每日科普短文

當觸發此 skill 時，執行以下步驟。**只處理一篇筆記，不要總結多篇。**

## 步驟

1. 執行腳本隨機選一篇筆記：
   ```
   bash /home/pi/.picoclaw/workspace/knowledge-review/pick-note.sh
   ```
   腳本會輸出一個檔案路徑。

2. 讀取該檔案的完整內容。

3. 根據內容改寫成一篇 300-500 字繁體中文科普短文，格式如下：

```
📚 今日知識複習

【標題】

引言（1-2 句帶出主題）

重點摘要（條列或短段落）

💡 延伸思考
一個引發好奇心的問題

📖 來源筆記：檔名
```

4. 將結果回覆給用戶。

5. 記錄已發送，執行：
   ```
   echo "$(date +%Y-%m-%d)|檔名|資料夾名" >> /home/pi/.picoclaw/workspace/knowledge-review/sent.log
   ```
