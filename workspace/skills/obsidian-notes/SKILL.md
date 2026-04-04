---
name: obsidian-notes
description: "Save messages, summaries, or images to the Obsidian vault as notes. Trigger: 筆記、note、記錄、摘要、總結、存檔、memo、save to notes、幫我記、寫入筆記。"
---

# Obsidian Notes

When the user asks to create a note, save a summary, or store content to their notebook, you MUST actually write a file to disk. Do NOT just say you saved it — you must use a tool to create the file.

## Vault Location

`/home/pi/Obsidian-Vault/`

## How to Create a Note

You MUST use the `write_file` tool to create a real Markdown file. Example tool call:

```
Tool: write_file
Arguments: {"path": "/home/pi/Obsidian-Vault/Inbox/2026-03-28-標題.md", "content": "---\ndate: 2026-03-28\nsource: line\ntags:\n  - from-line\n---\n\n# 標題\n\n內容"}
```

IMPORTANT: Always call the write_file tool. Never respond with just text claiming you saved a note.

### File naming

- Format: `YYYY-MM-DD-簡短標題.md`
- Default subfolder: `Inbox/`
- Use a different subfolder only if the user specifies

### Note frontmatter

```yaml
---
date: YYYY-MM-DD
source: line
tags:
  - from-line
  - <topic-tags>
---
```

### Content guidelines

- Summarize the user's message clearly and concisely
- If the user says "幫我建立總結" or "摘要", create a structured summary
- If the user sends raw text to save, preserve it with light formatting
- Use `[[wiki-link]]` syntax for cross-references when appropriate
- Write in the same language as the user's message

## Handling Images

When the user sends an image and asks to save it as a note:

1. **Describe the image** in the note body (you can see the image content)
2. **Copy the original file** to the vault using `execute_command`:

```
Tool: execute_command
Arguments: {"command": "bash /home/pi/.picoclaw/workspace/skills/obsidian-notes/scripts/save-media.sh \"<source_path>\" \"<title>\""}
```

The script outputs Obsidian embed syntax like `![[attachments/2026-03-28-photo.jpg]]`. Include this in the note.

3. If the source media path is unavailable, just save the text description of the image.

## Examples

User: "幫我把這段話建立成筆記：今天和老師討論了直翅目的分類方法"
→ Call write_file to create `/home/pi/Obsidian-Vault/Inbox/2026-03-28-直翅目分類討論.md`

User: [sends image] "存到筆記"
→ Call execute_command to copy image, then call write_file to create the note with image embed
