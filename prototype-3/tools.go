package main

import (
	"database/sql"
	"fmt"
	"strings"
)

// Bulk operations types

type ProposedAction struct {
	Description string `json:"description"`
	SQL         string `json:"sql"`
}

type ProposeActionsInput struct {
	Actions []ProposedAction `json:"actions"`
	Summary string           `json:"summary"`
}

// executeDML runs a single DML statement within a transaction.
// It whitelists UPDATE/INSERT/DELETE and blocks DDL keywords.
func executeDML(tx *sql.Tx, query string) (int64, error) {
	query = strings.TrimSpace(query)
	upper := strings.ToUpper(query)

	// Must start with a DML verb
	if !strings.HasPrefix(upper, "UPDATE") &&
		!strings.HasPrefix(upper, "INSERT") &&
		!strings.HasPrefix(upper, "DELETE") {
		return 0, fmt.Errorf("only UPDATE, INSERT, and DELETE statements are allowed")
	}

	// Block DDL keywords anywhere in the statement
	for _, kw := range []string{"DROP", "ALTER", "CREATE", "ATTACH", "DETACH"} {
		if strings.Contains(upper, kw+" ") || strings.HasSuffix(upper, kw) {
			return 0, fmt.Errorf("%s operations are not allowed", kw)
		}
	}

	result, err := tx.Exec(query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// QueryResult is the structured representation of SQL results,
// used for both SSE sql_result events and widget data injection.
type QueryResult struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
}

// Tool input types

type ExecuteSQLInput struct {
	Query string `json:"query"`
}

type CreateWidgetInput struct {
	Title   string `json:"title"`
	Slug    string `json:"slug"`
	Context string `json:"context"`
	SQL     string `json:"sql"`
	HTML    string `json:"html"`
}

type CreateDashboardInput struct {
	Title       string   `json:"title"`
	Slug        string   `json:"slug"`
	WidgetSlugs []string `json:"widget_slugs"`
}

type ReturnOrderIDsInput struct {
	OrderIDs    []int  `json:"order_ids"`
	Explanation string `json:"explanation"`
}

// executeSQL returns pipe-delimited text results for the agent.
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

// executeSQLStructured returns a QueryResult for SSE events and widget data injection.
func executeSQLStructured(db *sql.DB, query string) (*QueryResult, error) {
	query = strings.TrimSpace(query)

	upper := strings.ToUpper(query)
	if !strings.HasPrefix(upper, "SELECT") && !strings.HasPrefix(upper, "WITH") && !strings.HasPrefix(upper, "EXPLAIN") {
		return nil, fmt.Errorf("only SELECT, WITH, and EXPLAIN queries are allowed")
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("SQL error: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("reading columns: %w", err)
	}

	result := &QueryResult{Columns: cols}
	limit := 200

	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if len(result.Rows) >= limit {
			break
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		row := make([]interface{}, len(cols))
		for i, v := range values {
			// Convert []byte to string for JSON serialization
			if b, ok := v.([]byte); ok {
				row[i] = string(b)
			} else {
				row[i] = v
			}
		}
		result.Rows = append(result.Rows, row)
	}

	return result, nil
}
