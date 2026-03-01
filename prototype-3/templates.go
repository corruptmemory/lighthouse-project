package main

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"strings"
	"time"
)

type Templates struct {
	layout      *template.Template
	orderTable  *template.Template
	orderDetail *template.Template
	orderForm   *template.Template
	dashboard   *template.Template
}

var funcMap = template.FuncMap{
	"formatDate": func(s string) string {
		for _, layout := range []string{"2006-01-02", "2006-01-02T15:04:05Z", time.RFC3339} {
			if t, err := time.Parse(layout, s); err == nil {
				return t.Format("Jan 2, 2006")
			}
		}
		return s
	},
	"formatMoney": func(f float64) string {
		return fmt.Sprintf("$%.2f", f)
	},
	"inputDate": func(s string) string {
		for _, layout := range []string{"2006-01-02", "2006-01-02T15:04:05Z", time.RFC3339} {
			if t, err := time.Parse(layout, s); err == nil {
				return t.Format("2006-01-02")
			}
		}
		return s
	},
	"lower": strings.ToLower,
	"sortIcon": func(column, sortCol, sortDir string) template.HTML {
		if column != sortCol {
			return template.HTML(`<span class="sort-icon">⇕</span>`)
		}
		if sortDir == "asc" {
			return template.HTML(`<span class="sort-icon active">▲</span>`)
		}
		return template.HTML(`<span class="sort-icon active">▼</span>`)
	},
	"nextDir": func(column, sortCol, sortDir string) string {
		if column == sortCol && sortDir == "asc" {
			return "desc"
		}
		return "asc"
	},
	"statusClass": func(status string) string {
		switch status {
		case "Processing":
			return "status-processing"
		case "Shipped":
			return "status-shipped"
		case "Delivered":
			return "status-delivered"
		case "Return Requested":
			return "status-return"
		default:
			return ""
		}
	},
	"brandClass": func(brand string) string {
		switch brand {
		case "Leuchtturm1917":
			return "brand-lt"
		case "Lighthouse Publications":
			return "brand-lp"
		case "Mixed":
			return "brand-mixed"
		default:
			return ""
		}
	},
}

func LoadTemplates() (*Templates, error) {
	dir := "templates"

	layout, err := template.New("layout.html").Funcs(funcMap).ParseFiles(
		filepath.Join(dir, "layout.html"),
		filepath.Join(dir, "order-table.html"),
	)
	if err != nil {
		return nil, fmt.Errorf("parse layout: %w", err)
	}

	orderTable, err := template.New("order-table.html").Funcs(funcMap).ParseFiles(
		filepath.Join(dir, "order-table.html"),
	)
	if err != nil {
		return nil, fmt.Errorf("parse order-table: %w", err)
	}

	orderDetail, err := template.New("order-detail.html").Funcs(funcMap).ParseFiles(
		filepath.Join(dir, "order-detail.html"),
	)
	if err != nil {
		return nil, fmt.Errorf("parse order-detail: %w", err)
	}

	orderForm, err := template.New("order-form.html").Funcs(funcMap).ParseFiles(
		filepath.Join(dir, "order-form.html"),
	)
	if err != nil {
		return nil, fmt.Errorf("parse order-form: %w", err)
	}

	dashboard, err := template.New("dashboard.html").Funcs(funcMap).ParseFiles(
		filepath.Join(dir, "dashboard.html"),
	)
	if err != nil {
		return nil, fmt.Errorf("parse dashboard: %w", err)
	}

	return &Templates{
		layout:      layout,
		orderTable:  orderTable,
		orderDetail: orderDetail,
		orderForm:   orderForm,
		dashboard:   dashboard,
	}, nil
}

func (t *Templates) RenderLayout(w io.Writer, data any) error {
	return t.layout.ExecuteTemplate(w, "layout.html", data)
}

func (t *Templates) RenderOrderTable(w io.Writer, data any) error {
	return t.orderTable.ExecuteTemplate(w, "order-table-partial", data)
}

func (t *Templates) RenderOrderDetail(w io.Writer, data any) error {
	return t.orderDetail.ExecuteTemplate(w, "order-detail.html", data)
}

func (t *Templates) RenderOrderForm(w io.Writer, data any) error {
	return t.orderForm.ExecuteTemplate(w, "order-form.html", data)
}

func (t *Templates) RenderDashboard(w io.Writer) error {
	return t.dashboard.ExecuteTemplate(w, "dashboard.html", nil)
}
