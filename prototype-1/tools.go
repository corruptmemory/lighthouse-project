package main

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

// Widget storage
var (
	widgets   sync.Map
	widgetSeq atomic.Int64
)

// Tool input types

type ExecuteSQLInput struct {
	Query string `json:"query"`
}

type RenderHTMLInput struct {
	Title string `json:"title"`
	HTML  string `json:"html"`
}

// Tool executors

func executeSQL(db *sql.DB, input ExecuteSQLInput) (string, bool) {
	query := strings.TrimSpace(input.Query)

	// Validate read-only
	upper := strings.ToUpper(query)
	if !strings.HasPrefix(upper, "SELECT") && !strings.HasPrefix(upper, "WITH") && !strings.HasPrefix(upper, "EXPLAIN") {
		return "Error: only SELECT, WITH, and EXPLAIN queries are allowed.", true
	}

	// Check for dangerous keywords
	for _, kw := range []string{"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE", "ATTACH", "DETACH"} {
		if strings.Contains(upper, kw+" ") || strings.HasSuffix(upper, kw) {
			return fmt.Sprintf("Error: %s operations are not allowed. Read-only queries only.", kw), true
		}
	}

	rows, err := db.Query(query)
	if err != nil {
		return fmt.Sprintf("SQL Error: %v", err), true
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return fmt.Sprintf("Error reading columns: %v", err), true
	}

	var sb strings.Builder
	sb.WriteString(strings.Join(cols, " | "))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("-", len(sb.String())-1))
	sb.WriteString("\n")

	rowCount := 0
	limit := 200

	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if rowCount >= limit {
			sb.WriteString(fmt.Sprintf("\n... (truncated at %d rows)", limit))
			break
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Sprintf("Error scanning row: %v", err), true
		}

		parts := make([]string, len(cols))
		for i, v := range values {
			if v == nil {
				parts[i] = "NULL"
			} else {
				parts[i] = fmt.Sprintf("%v", v)
			}
		}
		sb.WriteString(strings.Join(parts, " | "))
		sb.WriteString("\n")
		rowCount++
	}

	if rowCount == 0 {
		sb.WriteString("(no rows)\n")
	} else {
		sb.WriteString(fmt.Sprintf("\n(%d rows)\n", rowCount))
	}

	return sb.String(), false
}

func renderHTML(input RenderHTMLInput) (string, string, bool) {
	id := fmt.Sprintf("w%d", widgetSeq.Add(1))
	widgets.Store(id, input.HTML)
	return id, input.Title, false
}

func getWidget(id string) (string, bool) {
	v, ok := widgets.Load(id)
	if !ok {
		return "", false
	}
	return v.(string), true
}
