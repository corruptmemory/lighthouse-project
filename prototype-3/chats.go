package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// ChatSessionInfo describes a persisted chat session for the recovery UI.
type ChatSessionInfo struct {
	ID           string `json:"id"`
	Exchanges    int    `json:"exchanges"`
	FirstPrompt  string `json:"first_prompt"`
	LastModified string `json:"last_modified"`
}

// ChatExchangeSummary is a prompt + response pair for restoring visible chat history.
type ChatExchangeSummary struct {
	Prompt   string `json:"prompt"`
	Response string `json:"response"`
}

// listChatSessions scans the chats directory for session subdirectories and
// returns summary info for each, sorted by last-modified descending.
func listChatSessions(dir string) ([]ChatSessionInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read chats dir: %w", err)
	}

	var sessions []ChatSessionInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		sessionDir := filepath.Join(dir, e.Name())
		files, err := os.ReadDir(sessionDir)
		if err != nil {
			continue
		}

		var mdFiles []string
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") {
				mdFiles = append(mdFiles, f.Name())
			}
		}
		if len(mdFiles) == 0 {
			continue
		}

		sort.Strings(mdFiles)

		// Read first file for preview prompt
		firstPrompt := ""
		data, err := os.ReadFile(filepath.Join(sessionDir, mdFiles[0]))
		if err == nil {
			prompt, _, parseErr := parseChatMarkdown(string(data))
			if parseErr == nil {
				firstPrompt = prompt
			}
		}

		// Get directory mod time
		info, err := e.Info()
		var modTime time.Time
		if err == nil {
			modTime = info.ModTime()
		}

		sessions = append(sessions, ChatSessionInfo{
			ID:           e.Name(),
			Exchanges:    len(mdFiles),
			FirstPrompt:  firstPrompt,
			LastModified: modTime.UTC().Format(time.RFC3339),
		})
	}

	// Sort by last-modified descending
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastModified > sessions[j].LastModified
	})

	return sessions, nil
}

// loadChatSummaries reads all exchange files for a session and extracts
// prompt + last assistant text for each, suitable for restoring the visible chat.
func loadChatSummaries(dir, sessionID string) ([]ChatExchangeSummary, error) {
	sessionDir := filepath.Join(dir, sessionID)

	matches, err := filepath.Glob(filepath.Join(sessionDir, "*.md"))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}

	sort.Strings(matches)

	var summaries []ChatExchangeSummary
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		prompt, messages, err := parseChatMarkdown(string(data))
		if err != nil {
			continue
		}

		// Find the last assistant text message
		response := ""
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == anthropic.MessageParamRoleAssistant {
				response = extractLastText(messages[i])
				if response != "" {
					break
				}
			}
		}

		summaries = append(summaries, ChatExchangeSummary{
			Prompt:   prompt,
			Response: response,
		})
	}

	return summaries, nil
}

// extractLastText pulls the last text block from a message's content.
func extractLastText(msg anthropic.MessageParam) string {
	// MessageParam.Content is a union — marshal and re-parse to extract text blocks.
	raw, err := json.Marshal(msg.Content)
	if err != nil {
		return ""
	}

	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}

	lastText := ""
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			lastText = b.Text
		}
	}
	return lastText
}

// writeChatExchange writes a single chat exchange (prompt + messages) as a
// numbered markdown file in chats/{sessionID}/.
func writeChatExchange(dir, sessionID, prompt string, messages []anthropic.MessageParam) error {
	sessionDir := filepath.Join(dir, sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}

	// Count existing files to determine next sequence number
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return fmt.Errorf("read session dir: %w", err)
	}
	seq := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			seq++
		}
	}

	messagesJSON, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal messages: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("## Prompt\n")
	sb.WriteString(prompt)
	sb.WriteString("\n\n## Messages\n```json\n")
	sb.Write(messagesJSON)
	sb.WriteString("\n```\n")

	filename := fmt.Sprintf("%06d.md", seq+1)
	path := filepath.Join(sessionDir, filename)
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

// loadChatHistory reads all exchange files for a session and concatenates
// their messages into a single slice, in file-order.
func loadChatHistory(dir, sessionID string) ([]anthropic.MessageParam, error) {
	sessionDir := filepath.Join(dir, sessionID)

	matches, err := filepath.Glob(filepath.Join(sessionDir, "*.md"))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}

	sort.Strings(matches) // lexicographic = numeric order

	var all []anthropic.MessageParam
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", filepath.Base(path), err)
		}

		_, messages, err := parseChatMarkdown(string(data))
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
		}
		all = append(all, messages...)
	}

	return all, nil
}

// parseChatMarkdown extracts the prompt text and messages JSON from a chat
// exchange markdown file.
func parseChatMarkdown(content string) (string, []anthropic.MessageParam, error) {
	// Extract prompt: text between "## Prompt" and the next "##"
	promptRe := regexp.MustCompile(`(?ms)^## Prompt\n(.+?)^##`)
	promptMatch := promptRe.FindStringSubmatch(content)
	if len(promptMatch) < 2 {
		return "", nil, fmt.Errorf("missing ## Prompt section")
	}
	prompt := strings.TrimSpace(promptMatch[1])

	// Extract messages JSON from fenced code block under ## Messages
	messagesRe := regexp.MustCompile("(?ms)^## Messages\\n```json\\n(.+?)```")
	messagesMatch := messagesRe.FindStringSubmatch(content)
	if len(messagesMatch) < 2 {
		return "", nil, fmt.Errorf("missing ## Messages JSON block")
	}

	var messages []anthropic.MessageParam
	if err := json.Unmarshal([]byte(messagesMatch[1]), &messages); err != nil {
		return "", nil, fmt.Errorf("unmarshal messages: %w", err)
	}

	return prompt, messages, nil
}
