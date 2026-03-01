package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type AgentConfig struct {
	APIKey     string
	DB         *sql.DB
	WidgetsDir string
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
			Description: anthropic.String("Execute a read-only SQL SELECT query against the SQLite database. Returns results as pipe-delimited text. Limited to 200 rows. The user will also see the query and results displayed inline in the chat."),
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
			Name:        "create_widget",
			Description: anthropic.String("Create a persistent widget that displays data from a SQL query. The widget is saved as a markdown file and can be loaded on any dashboard. IMPORTANT: The HTML must read data from `window.WIDGET_DATA` (which the server injects at render time) instead of hardcoding query results. The WIDGET_DATA object has the shape: {columns: string[], rows: any[][]}"),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"title": map[string]interface{}{
						"type":        "string",
						"description": "A short, descriptive title for the widget",
					},
					"slug": map[string]interface{}{
						"type":        "string",
						"description": "URL-safe identifier (lowercase, hyphens only). Example: revenue-by-brand-feb-2026",
					},
					"context": map[string]interface{}{
						"type":        "string",
						"description": "Brief description of what the user asked for and how the query answers it",
					},
					"sql": map[string]interface{}{
						"type":        "string",
						"description": "The SQL SELECT query that produces the data for this widget",
					},
					"html": map[string]interface{}{
						"type":        "string",
						"description": "Complete self-contained HTML document. Must use window.WIDGET_DATA for data. Include inline CSS/JS. No external resources (sandboxed iframe).",
					},
				},
				Required: []string{"title", "slug", "context", "sql", "html"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "list_widgets",
			Description: anthropic.String("List all saved widgets with their titles and slugs. Use this to check what widgets already exist before creating new ones or when composing dashboards."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "create_dashboard",
			Description: anthropic.String("Create a dashboard that displays multiple widgets in a grid layout. The dashboard is saved as a markdown file. Reference widgets by their slugs."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"title": map[string]interface{}{
						"type":        "string",
						"description": "A short, descriptive title for the dashboard",
					},
					"slug": map[string]interface{}{
						"type":        "string",
						"description": "URL-safe identifier for the dashboard (lowercase, hyphens only)",
					},
					"widget_slugs": map[string]interface{}{
						"type":        "array",
						"description": "Ordered list of widget slugs to include in the dashboard",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
				Required: []string{"title", "slug", "widget_slugs"},
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
				result, isError := dispatchTool(cfg, block.Name, block.Input, emit)

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

func dispatchTool(cfg AgentConfig, name string, input json.RawMessage, emit EmitFunc) (string, bool) {
	switch name {
	case "execute_sql":
		var inp ExecuteSQLInput
		if err := json.Unmarshal(input, &inp); err != nil {
			return fmt.Sprintf("Error parsing input: %v", err), true
		}
		log.Printf("[tool] execute_sql: %s", inp.Query)

		// Get pipe-delimited result for the agent
		result, isError := executeSQL(cfg.DB, inp)

		// Also emit structured result for inline display (even on error, we show what was attempted)
		if !isError {
			if structured, err := executeSQLStructured(cfg.DB, inp.Query); err == nil {
				emit("sql_result", map[string]interface{}{
					"query":     inp.Query,
					"columns":   structured.Columns,
					"rows":      structured.Rows,
					"row_count": len(structured.Rows),
				})
			}
		}

		return result, isError

	case "create_widget":
		var inp CreateWidgetInput
		if err := json.Unmarshal(input, &inp); err != nil {
			return fmt.Sprintf("Error parsing input: %v", err), true
		}
		log.Printf("[tool] create_widget: %s (%s)", inp.Title, inp.Slug)

		widget := &ParsedWidget{
			Title:     inp.Title,
			Slug:      inp.Slug,
			Generated: time.Now().Format("2006-01-02 15:04"),
			Context:   inp.Context,
			SQL:       inp.SQL,
			HTML:      inp.HTML,
		}

		if err := writeWidgetMarkdown(cfg.WidgetsDir, widget); err != nil {
			return fmt.Sprintf("Error saving widget: %v", err), true
		}

		emit("widget", map[string]string{
			"slug":  inp.Slug,
			"title": inp.Title,
		})

		return fmt.Sprintf("Widget '%s' saved as %s. The user can see it in their dashboard.", inp.Title, inp.Slug), false

	case "list_widgets":
		log.Printf("[tool] list_widgets")

		widgets, err := listWidgets(cfg.WidgetsDir)
		if err != nil {
			return fmt.Sprintf("Error listing widgets: %v", err), true
		}

		if len(widgets) == 0 {
			return "No widgets saved yet.", false
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d widget(s):\n", len(widgets)))
		for _, w := range widgets {
			sb.WriteString(fmt.Sprintf("- %s (slug: %s)\n", w.Title, w.Slug))
		}
		return sb.String(), false

	case "create_dashboard":
		var inp CreateDashboardInput
		if err := json.Unmarshal(input, &inp); err != nil {
			return fmt.Sprintf("Error parsing input: %v", err), true
		}
		log.Printf("[tool] create_dashboard: %s (%s) with %d widgets", inp.Title, inp.Slug, len(inp.WidgetSlugs))

		dashboard := &ParsedDashboard{
			Title:       inp.Title,
			Slug:        inp.Slug,
			WidgetSlugs: inp.WidgetSlugs,
		}

		if err := writeDashboardMarkdown(cfg.WidgetsDir, dashboard); err != nil {
			return fmt.Sprintf("Error saving dashboard: %v", err), true
		}

		emit("dashboard", map[string]interface{}{
			"slug":         inp.Slug,
			"title":        inp.Title,
			"widget_slugs": inp.WidgetSlugs,
		})

		return fmt.Sprintf("Dashboard '%s' saved. View it at /api/dashboards/%s", inp.Title, inp.Slug), false

	default:
		return fmt.Sprintf("Unknown tool: %s", name), true
	}
}

func runFilterAgentLoop(ctx context.Context, cfg AgentConfig, prompt string) (*ReturnOrderIDsInput, error) {
	client := anthropic.NewClient(option.WithAPIKey(cfg.APIKey))

	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
	}

	systemPrompt := buildFilterSystemPrompt()

	tools := []anthropic.ToolUnionParam{
		{OfTool: &anthropic.ToolParam{
			Name:        "execute_sql",
			Description: anthropic.String("Execute a read-only SQL SELECT query against the SQLite database. Returns results as pipe-delimited text."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "A read-only SQL SELECT query",
					},
				},
				Required: []string{"query"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "return_order_ids",
			Description: anthropic.String("Return the final list of matching order IDs. Call this when you have identified all orders matching the user's request."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"order_ids": map[string]interface{}{
						"type":        "array",
						"description": "Array of order_id integers from the orders table",
						"items":       map[string]interface{}{"type": "integer"},
					},
					"explanation": map[string]interface{}{
						"type":        "string",
						"description": "Brief explanation of why these orders match",
					},
				},
				Required: []string{"order_ids", "explanation"},
			},
		}},
	}

	maxIterations := 15
	for i := 0; i < maxIterations; i++ {
		resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 4096,
			System: []anthropic.TextBlockParam{
				{Text: systemPrompt},
			},
			Messages: messages,
			Tools:    tools,
		})
		if err != nil {
			return nil, fmt.Errorf("anthropic API: %w", err)
		}

		var assistantBlocks []anthropic.ContentBlockParamUnion
		var toolResultBlocks []anthropic.ContentBlockParamUnion
		hasToolUse := false

		for _, block := range resp.Content {
			switch block.Type {
			case "text":
				assistantBlocks = append(assistantBlocks, anthropic.NewTextBlock(block.Text))
			case "thinking":
				assistantBlocks = append(assistantBlocks, anthropic.NewThinkingBlock(block.Signature, block.Thinking))
			case "tool_use":
				hasToolUse = true
				assistantBlocks = append(assistantBlocks, anthropic.NewToolUseBlock(block.ID, block.Input, block.Name))

				if block.Name == "return_order_ids" {
					var result ReturnOrderIDsInput
					inputJSON, _ := json.Marshal(block.Input)
					if err := json.Unmarshal(inputJSON, &result); err != nil {
						return nil, fmt.Errorf("parse return_order_ids: %w", err)
					}
					log.Printf("[filter] return_order_ids: %v — %s", result.OrderIDs, result.Explanation)
					return &result, nil
				}

				if block.Name == "execute_sql" {
					var inp ExecuteSQLInput
					inputJSON, _ := json.Marshal(block.Input)
					if err := json.Unmarshal(inputJSON, &inp); err != nil {
						toolResultBlocks = append(toolResultBlocks, anthropic.NewToolResultBlock(block.ID, "Error parsing input", true))
						continue
					}
					log.Printf("[filter] execute_sql: %s", inp.Query)
					result, isError := executeSQL(cfg.DB, inp)
					toolResultBlocks = append(toolResultBlocks, anthropic.NewToolResultBlock(block.ID, result, isError))
				}
			}
		}

		messages = append(messages, anthropic.NewAssistantMessage(assistantBlocks...))

		if hasToolUse && len(toolResultBlocks) > 0 {
			messages = append(messages, anthropic.NewUserMessage(toolResultBlocks...))
			continue
		}

		// No tool use — model finished without calling return_order_ids
		if resp.StopReason == anthropic.StopReasonEndTurn || resp.StopReason == "" {
			break
		}
	}

	return nil, fmt.Errorf("agent did not return order IDs within %d iterations", maxIterations)
}

func runBulkOpsAgentLoop(ctx context.Context, cfg AgentConfig, prompt string) (*ProposeActionsInput, error) {
	client := anthropic.NewClient(option.WithAPIKey(cfg.APIKey))

	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
	}

	systemPrompt := buildBulkOpsSystemPrompt()

	tools := []anthropic.ToolUnionParam{
		{OfTool: &anthropic.ToolParam{
			Name:        "execute_sql",
			Description: anthropic.String("Execute a read-only SQL SELECT query against the SQLite database. Returns results as pipe-delimited text."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "A read-only SQL SELECT query",
					},
				},
				Required: []string{"query"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "propose_actions",
			Description: anthropic.String("Propose a set of DML actions to execute. Call this when you have determined all the changes needed. Each action has a human-readable description and a single SQL DML statement."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"actions": map[string]interface{}{
						"type":        "array",
						"description": "Array of proposed DML actions",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"description": map[string]interface{}{
									"type":        "string",
									"description": "Human-readable description of what this action does, with concrete values",
								},
								"sql": map[string]interface{}{
									"type":        "string",
									"description": "A single UPDATE, INSERT, or DELETE SQL statement",
								},
							},
							"required": []string{"description", "sql"},
						},
					},
					"summary": map[string]interface{}{
						"type":        "string",
						"description": "Brief summary of the overall operation and any assumptions made",
					},
				},
				Required: []string{"actions", "summary"},
			},
		}},
	}

	maxIterations := 15
	for i := 0; i < maxIterations; i++ {
		resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 4096,
			System: []anthropic.TextBlockParam{
				{Text: systemPrompt},
			},
			Messages: messages,
			Tools:    tools,
		})
		if err != nil {
			return nil, fmt.Errorf("anthropic API: %w", err)
		}

		var assistantBlocks []anthropic.ContentBlockParamUnion
		var toolResultBlocks []anthropic.ContentBlockParamUnion
		hasToolUse := false

		for _, block := range resp.Content {
			switch block.Type {
			case "text":
				assistantBlocks = append(assistantBlocks, anthropic.NewTextBlock(block.Text))
			case "thinking":
				assistantBlocks = append(assistantBlocks, anthropic.NewThinkingBlock(block.Signature, block.Thinking))
			case "tool_use":
				hasToolUse = true
				assistantBlocks = append(assistantBlocks, anthropic.NewToolUseBlock(block.ID, block.Input, block.Name))

				if block.Name == "propose_actions" {
					var result ProposeActionsInput
					inputJSON, _ := json.Marshal(block.Input)
					if err := json.Unmarshal(inputJSON, &result); err != nil {
						return nil, fmt.Errorf("parse propose_actions: %w", err)
					}
					log.Printf("[bulk-ops] propose_actions: %d actions — %s", len(result.Actions), result.Summary)
					return &result, nil
				}

				if block.Name == "execute_sql" {
					var inp ExecuteSQLInput
					inputJSON, _ := json.Marshal(block.Input)
					if err := json.Unmarshal(inputJSON, &inp); err != nil {
						toolResultBlocks = append(toolResultBlocks, anthropic.NewToolResultBlock(block.ID, "Error parsing input", true))
						continue
					}
					log.Printf("[bulk-ops] execute_sql: %s", inp.Query)
					result, isError := executeSQL(cfg.DB, inp)
					toolResultBlocks = append(toolResultBlocks, anthropic.NewToolResultBlock(block.ID, result, isError))
				}
			}
		}

		messages = append(messages, anthropic.NewAssistantMessage(assistantBlocks...))

		if hasToolUse && len(toolResultBlocks) > 0 {
			messages = append(messages, anthropic.NewUserMessage(toolResultBlocks...))
			continue
		}

		if resp.StopReason == anthropic.StopReasonEndTurn || resp.StopReason == "" {
			break
		}
	}

	return nil, fmt.Errorf("agent did not propose actions within %d iterations", maxIterations)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
