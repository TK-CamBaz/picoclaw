package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// ClaudeCliProvider implements LLMProvider using the claude CLI as a subprocess.
type ClaudeCliProvider struct {
	command   string
	workspace string
}

// NewClaudeCliProvider creates a new Claude CLI provider.
func NewClaudeCliProvider(workspace string) *ClaudeCliProvider {
	return &ClaudeCliProvider{
		command:   "claude",
		workspace: workspace,
	}
}

// Chat implements LLMProvider.Chat by executing the claude CLI.
func (p *ClaudeCliProvider) Chat(
	ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]any,
) (*LLMResponse, error) {
	if hasMedia(messages) {
		return p.chatWithStreamJSON(ctx, messages, tools, model)
	}

	systemPrompt := p.buildSystemPrompt(messages, nil)
	prompt := p.messagesToPrompt(messages)

	args := []string{"-p", "--output-format", "json", "--dangerously-skip-permissions", "--no-chrome"}
	if systemPrompt != "" {
		args = append(args, "--system-prompt", systemPrompt)
	}
	if model != "" && model != "claude-code" {
		args = append(args, "--model", model)
	}
	args = append(args, "-") // read from stdin

	cmd := exec.CommandContext(ctx, p.command, args...)
	if p.workspace != "" {
		cmd.Dir = p.workspace
	}
	cmd.Stdin = bytes.NewReader([]byte(prompt))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		stdoutStr := strings.TrimSpace(stdout.String())
		switch {
		case stderrStr != "" && stdoutStr != "":
			return nil, fmt.Errorf("claude cli error: %w\nstderr: %s\nstdout: %s", err, stderrStr, stdoutStr)
		case stderrStr != "":
			return nil, fmt.Errorf("claude cli error: %s", stderrStr)
		case stdoutStr != "":
			return nil, fmt.Errorf("claude cli error: %w\noutput: %s", err, stdoutStr)
		default:
			return nil, fmt.Errorf("claude cli error: %w", err)
		}
	}

	return p.parseClaudeCliResponse(stdout.String())
}

// hasMedia returns true if any message contains media attachments.
func hasMedia(messages []Message) bool {
	for _, m := range messages {
		if len(m.Media) > 0 {
			return true
		}
	}
	return false
}

// buildStreamJSONInput serializes messages into a stream-json input line for claude CLI.
// The last user message's Media are included as image content blocks.
func (p *ClaudeCliProvider) buildStreamJSONInput(messages []Message) ([]byte, error) {
	prompt := p.messagesToPrompt(messages)

	// Collect media from the last user message
	var mediaURLs []string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" && len(messages[i].Media) > 0 {
			mediaURLs = messages[i].Media
			break
		}
	}

	type textBlock struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type imageSource struct {
		Type      string `json:"type"`
		MediaType string `json:"media_type"`
		Data      string `json:"data"`
	}
	type imageBlock struct {
		Type   string      `json:"type"`
		Source imageSource `json:"source"`
	}
	type messageContent struct {
		Role    string `json:"role"`
		Content []any  `json:"content"`
	}
	type streamJSONInput struct {
		Type    string         `json:"type"`
		Message messageContent `json:"message"`
	}

	content := []any{textBlock{Type: "text", Text: prompt}}

	for _, mediaURL := range mediaURLs {
		parts := strings.SplitN(mediaURL, ",", 2)
		if len(parts) != 2 {
			log.Printf("claude_cli_provider: skipping malformed media URL (no comma): %q", mediaURL[:min(len(mediaURL), 80)])
			continue
		}
		header := strings.TrimPrefix(parts[0], "data:")
		headerParts := strings.SplitN(header, ";", 2)
		if len(headerParts) != 2 || headerParts[1] != "base64" {
			log.Printf("claude_cli_provider: skipping malformed media URL (bad header): %q", parts[0])
			continue
		}
		mediaType := headerParts[0]
		b64data := parts[1]
		content = append(content, imageBlock{
			Type: "image",
			Source: imageSource{
				Type:      "base64",
				MediaType: mediaType,
				Data:      b64data,
			},
		})
	}

	input := streamJSONInput{
		Type: "user",
		Message: messageContent{
			Role:    "user",
			Content: content,
		},
	}

	return json.Marshal(input)
}

// chatWithStreamJSON executes claude CLI in stream-json mode to support image inputs.
func (p *ClaudeCliProvider) chatWithStreamJSON(ctx context.Context, messages []Message, tools []ToolDefinition, model string) (*LLMResponse, error) {
	systemPrompt := p.buildSystemPrompt(messages, nil)

	inputJSON, err := p.buildStreamJSONInput(messages)
	if err != nil {
		return nil, fmt.Errorf("claude cli: failed to build stream-json input: %w", err)
	}

	args := []string{"-p", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose", "--dangerously-skip-permissions", "--no-chrome"}
	if systemPrompt != "" {
		args = append(args, "--system-prompt", systemPrompt)
	}
	if model != "" && model != "claude-code" {
		args = append(args, "--model", model)
	}

	cmd := exec.CommandContext(ctx, p.command, args...)
	if p.workspace != "" {
		cmd.Dir = p.workspace
	}
	cmd.Stdin = bytes.NewReader(inputJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderrStr := stderr.String(); stderrStr != "" {
			return nil, fmt.Errorf("claude cli error: %s", stderrStr)
		}
		return nil, fmt.Errorf("claude cli error: %w", err)
	}

	resultLine, err := parseStreamJSONOutput(stdout.String())
	if err != nil {
		return nil, err
	}

	return p.parseClaudeCliResponse(resultLine)
}

// parseStreamJSONOutput finds the result event line from stream-json output.
func parseStreamJSONOutput(output string) (string, error) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, `"type":"result"`) || strings.Contains(line, `"type": "result"`) {
			return line, nil
		}
	}
	return "", fmt.Errorf("claude cli: no result event found in stream-json output")
}

// min returns the smaller of a and b.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetDefaultModel returns the default model identifier.
func (p *ClaudeCliProvider) GetDefaultModel() string {
	return "claude-code"
}

// messagesToPrompt converts messages to a CLI-compatible prompt string.
func (p *ClaudeCliProvider) messagesToPrompt(messages []Message) string {
	var parts []string

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			// handled via --system-prompt flag
		case "user":
			parts = append(parts, "User: "+msg.Content)
		case "assistant":
			parts = append(parts, "Assistant: "+msg.Content)
		case "tool":
			parts = append(parts, fmt.Sprintf("[Tool Result for %s]: %s", msg.ToolCallID, msg.Content))
		}
	}

	// Simplify single user message
	if len(parts) == 1 && strings.HasPrefix(parts[0], "User: ") {
		return strings.TrimPrefix(parts[0], "User: ")
	}

	return strings.Join(parts, "\n")
}

// buildSystemPrompt combines system messages and tool definitions.
func (p *ClaudeCliProvider) buildSystemPrompt(messages []Message, tools []ToolDefinition) string {
	var parts []string

	for _, msg := range messages {
		if msg.Role == "system" {
			parts = append(parts, msg.Content)
		}
	}

	if len(tools) > 0 {
		parts = append(parts, buildCLIToolsPrompt(tools))
	}

	return strings.Join(parts, "\n\n")
}

// parseClaudeCliResponse parses the JSON output from the claude CLI.
func (p *ClaudeCliProvider) parseClaudeCliResponse(output string) (*LLMResponse, error) {
	var resp claudeCliJSONResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse claude cli response: %w", err)
	}

	if resp.IsError {
		return nil, fmt.Errorf("claude cli returned error: %s", resp.Result)
	}

	toolCalls := p.extractToolCalls(resp.Result)

	finishReason := "stop"
	content := resp.Result
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
		content = p.stripToolCallsJSON(resp.Result)
	}

	var usage *UsageInfo
	if resp.Usage.InputTokens > 0 || resp.Usage.OutputTokens > 0 {
		usage = &UsageInfo{
			PromptTokens:     resp.Usage.InputTokens + resp.Usage.CacheCreationInputTokens + resp.Usage.CacheReadInputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.CacheCreationInputTokens + resp.Usage.CacheReadInputTokens + resp.Usage.OutputTokens,
		}
	}

	return &LLMResponse{
		Content:      strings.TrimSpace(content),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage:        usage,
	}, nil
}

// extractToolCalls delegates to the shared extractToolCallsFromText function.
func (p *ClaudeCliProvider) extractToolCalls(text string) []ToolCall {
	return extractToolCallsFromText(text)
}

// stripToolCallsJSON delegates to the shared stripToolCallsFromText function.
func (p *ClaudeCliProvider) stripToolCallsJSON(text string) string {
	return stripToolCallsFromText(text)
}

// findMatchingBrace finds the index after the closing brace matching the opening brace at pos.
func findMatchingBrace(text string, pos int) int {
	depth := 0
	for i := pos; i < len(text); i++ {
		if text[i] == '{' {
			depth++
		} else if text[i] == '}' {
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return pos
}

// claudeCliJSONResponse represents the JSON output from the claude CLI.
// Matches the real claude CLI v2.x output format.
type claudeCliJSONResponse struct {
	Type         string             `json:"type"`
	Subtype      string             `json:"subtype"`
	IsError      bool               `json:"is_error"`
	Result       string             `json:"result"`
	SessionID    string             `json:"session_id"`
	TotalCostUSD float64            `json:"total_cost_usd"`
	DurationMS   int                `json:"duration_ms"`
	DurationAPI  int                `json:"duration_api_ms"`
	NumTurns     int                `json:"num_turns"`
	Usage        claudeCliUsageInfo `json:"usage"`
}

// claudeCliUsageInfo represents token usage from the claude CLI response.
type claudeCliUsageInfo struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}
