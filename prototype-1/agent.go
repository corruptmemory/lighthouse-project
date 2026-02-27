package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type AgentConfig struct {
	APIKey string
	DB     *sql.DB
}

type EmitFunc func(eventType string, data interface{})

func runAgentLoop(ctx context.Context, cfg AgentConfig, session *Session, userMessage string, emit EmitFunc) error {
	client := anthropic.NewClient(option.WithAPIKey(cfg.APIKey))

	// Append user message to session history
	session.Messages = append(session.Messages, anthropic.NewUserMessage(
		anthropic.NewTextBlock(userMessage),
	))

	systemPrompt := buildSystemPrompt()

	tools := []anthropic.ToolUnionParam{
		{OfTool: &anthropic.ToolParam{
			Name:        "execute_sql",
			Description: anthropic.String("Execute a read-only SQL SELECT query against the SQLite database. Returns results as pipe-delimited text. Limited to 200 rows."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "A read-only SQL SELECT query to execute against the database",
					},
				},
				Required: []string{"query"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "render_html",
			Description: anthropic.String("Render an HTML widget displayed in the user's dashboard. Create self-contained HTML with inline CSS/JS. No external resources (sandboxed iframe). Use design tokens from system prompt."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"title": map[string]interface{}{
						"type":        "string",
						"description": "A short title for the widget",
					},
					"html": map[string]interface{}{
						"type":        "string",
						"description": "Complete self-contained HTML document for the widget. Include inline CSS and JS. No external resources.",
					},
				},
				Required: []string{"title", "html"},
			},
		}},
	}

	// Agent loop — keep calling until end_turn or error
	maxIterations := 15
	for i := 0; i < maxIterations; i++ {
		emit("thinking", map[string]string{"status": "calling model..."})

		resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 8192,
			System: []anthropic.TextBlockParam{
				{Text: systemPrompt},
			},
			Messages: session.Messages,
			Tools:    tools,
		})
		if err != nil {
			return fmt.Errorf("anthropic API: %w", err)
		}

		// Process response content blocks
		var assistantBlocks []anthropic.ContentBlockParamUnion
		var toolResultBlocks []anthropic.ContentBlockParamUnion
		hasToolUse := false

		for _, block := range resp.Content {
			switch block.Type {
			case "text":
				assistantBlocks = append(assistantBlocks, anthropic.NewTextBlock(block.Text))
				emit("text", map[string]string{"text": block.Text})

			case "thinking":
				assistantBlocks = append(assistantBlocks, anthropic.NewThinkingBlock(block.Signature, block.Thinking))

			case "tool_use":
				hasToolUse = true

				assistantBlocks = append(assistantBlocks, anthropic.NewToolUseBlock(block.ID, block.Input, block.Name))

				inputJSON, _ := json.Marshal(block.Input)
				emit("tool_use", map[string]string{
					"tool_id": block.ID,
					"name":    block.Name,
					"input":   string(inputJSON),
				})

				// Dispatch tool
				result, isError := dispatchTool(cfg.DB, block.Name, block.Input, emit)

				toolResultBlocks = append(toolResultBlocks, anthropic.NewToolResultBlock(block.ID, result, isError))

				emit("tool_result", map[string]interface{}{
					"tool_id":  block.ID,
					"name":     block.Name,
					"result":   truncate(result, 2000),
					"is_error": isError,
				})
			}
		}

		// Add assistant message to history
		session.Messages = append(session.Messages, anthropic.NewAssistantMessage(assistantBlocks...))

		// If there were tool uses, add results and continue loop
		if hasToolUse {
			session.Messages = append(session.Messages, anthropic.NewUserMessage(toolResultBlocks...))
			continue
		}

		// No tool use — model is done
		if resp.StopReason == anthropic.StopReasonEndTurn || resp.StopReason == "" {
			break
		}
	}

	emit("done", nil)
	return nil
}

func dispatchTool(db *sql.DB, name string, input json.RawMessage, emit EmitFunc) (string, bool) {
	switch name {
	case "execute_sql":
		var inp ExecuteSQLInput
		if err := json.Unmarshal(input, &inp); err != nil {
			return fmt.Sprintf("Error parsing input: %v", err), true
		}
		log.Printf("[tool] execute_sql: %s", inp.Query)
		return executeSQL(db, inp)

	case "render_html":
		var inp RenderHTMLInput
		if err := json.Unmarshal(input, &inp); err != nil {
			return fmt.Sprintf("Error parsing input: %v", err), true
		}
		log.Printf("[tool] render_html: %s", inp.Title)
		widgetID, title, isError := renderHTML(inp)
		if isError {
			return "Error storing widget", true
		}
		emit("widget", map[string]string{
			"widget_id": widgetID,
			"title":     title,
		})
		return fmt.Sprintf("Widget created with ID: %s (title: %s). The user can now see it in their dashboard.", widgetID, title), false

	default:
		return fmt.Sprintf("Unknown tool: %s", name), true
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
