package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ParsedWidget represents a widget parsed from a markdown file.
type ParsedWidget struct {
	Title     string
	Slug      string
	Generated string
	Context   string
	SQL       string
	HTML      string
}

// parseWidgetMarkdown extracts structured data from a widget markdown file.
// Expected format:
//
//	# Title
//	Generated: ...
//	Slug: ...
//	## Context
//	...
//	## SQL
//	```sql
//	...
//	```
//	## Widget
//	```html
//	...
//	```
func parseWidgetMarkdown(content string) (*ParsedWidget, error) {
	w := &ParsedWidget{}

	// Extract title from first H1
	if m := regexp.MustCompile(`(?m)^# (.+)$`).FindStringSubmatch(content); len(m) > 1 {
		w.Title = strings.TrimSpace(m[1])
	}

	// Extract metadata lines
	if m := regexp.MustCompile(`(?m)^Generated:\s*(.+)$`).FindStringSubmatch(content); len(m) > 1 {
		w.Generated = strings.TrimSpace(m[1])
	}
	if m := regexp.MustCompile(`(?m)^Slug:\s*(.+)$`).FindStringSubmatch(content); len(m) > 1 {
		w.Slug = strings.TrimSpace(m[1])
	}

	// Extract context section (text between ## Context and the next ##)
	if m := regexp.MustCompile(`(?ms)^## Context\n(.+?)^##`).FindStringSubmatch(content); len(m) > 1 {
		w.Context = strings.TrimSpace(m[1])
	}

	// Extract SQL from fenced code block under ## SQL
	if m := regexp.MustCompile("(?ms)^## SQL\\n```sql\\n(.+?)```").FindStringSubmatch(content); len(m) > 1 {
		w.SQL = strings.TrimSpace(m[1])
	}

	// Extract HTML from fenced code block under ## Widget
	if m := regexp.MustCompile("(?ms)^## Widget\\n```html\\n(.+?)```").FindStringSubmatch(content); len(m) > 1 {
		w.HTML = strings.TrimSpace(m[1])
	}

	if w.SQL == "" {
		return nil, fmt.Errorf("widget markdown missing SQL section")
	}
	if w.HTML == "" {
		return nil, fmt.Errorf("widget markdown missing HTML section")
	}

	return w, nil
}

// writeWidgetMarkdown creates a widget markdown file on disk.
func writeWidgetMarkdown(dir string, w *ParsedWidget) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", w.Title))
	sb.WriteString(fmt.Sprintf("Generated: %s\n", w.Generated))
	sb.WriteString(fmt.Sprintf("Slug: %s\n\n", w.Slug))
	sb.WriteString(fmt.Sprintf("## Context\n%s\n\n", w.Context))
	sb.WriteString("## SQL\n```sql\n")
	sb.WriteString(w.SQL)
	sb.WriteString("\n```\n\n")
	sb.WriteString("## Widget\n```html\n")
	sb.WriteString(w.HTML)
	sb.WriteString("\n```\n")

	path := filepath.Join(dir, w.Slug+".widget.md")
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

// loadWidget reads and parses a widget markdown file by slug.
func loadWidget(dir, slug string) (*ParsedWidget, error) {
	path := filepath.Join(dir, slug+".widget.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read widget %q: %w", slug, err)
	}
	return parseWidgetMarkdown(string(data))
}

// listWidgets returns all widget slugs and titles from the widgets directory.
func listWidgets(dir string) ([]ParsedWidget, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.widget.md"))
	if err != nil {
		return nil, err
	}

	var widgets []ParsedWidget
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		w, err := parseWidgetMarkdown(string(data))
		if err != nil {
			continue
		}
		// Derive slug from filename if not set
		if w.Slug == "" {
			base := filepath.Base(path)
			w.Slug = strings.TrimSuffix(base, ".widget.md")
		}
		widgets = append(widgets, *w)
	}
	return widgets, nil
}

// renderWidget executes the widget's SQL and injects fresh data into the HTML.
func renderWidget(db *sql.DB, widget *ParsedWidget) (string, error) {
	result, err := executeSQLStructured(db, widget.SQL)
	if err != nil {
		return "", fmt.Errorf("execute widget SQL: %w", err)
	}

	dataJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal widget data: %w", err)
	}

	injection := fmt.Sprintf(`<script>window.WIDGET_DATA = %s;</script>`, dataJSON)

	html := widget.HTML
	if strings.Contains(html, "<body>") {
		html = strings.Replace(html, "<body>", "<body>"+injection, 1)
	} else if strings.Contains(html, "</head>") {
		html = strings.Replace(html, "</head>", injection+"</head>", 1)
	} else {
		// Prepend if no body/head tag found
		html = injection + html
	}

	return html, nil
}
