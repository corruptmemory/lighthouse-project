package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ParsedDashboard represents a dashboard parsed from a markdown file.
type ParsedDashboard struct {
	Title       string
	Slug        string
	WidgetSlugs []string
}

// parseDashboardMarkdown extracts structured data from a dashboard markdown file.
// Expected format:
//
//	# Title
//	Slug: ...
//	## Widgets
//	- widget-slug-1
//	- widget-slug-2
func parseDashboardMarkdown(content string) (*ParsedDashboard, error) {
	d := &ParsedDashboard{}

	// Extract title from first H1
	if m := regexp.MustCompile(`(?m)^# (.+)$`).FindStringSubmatch(content); len(m) > 1 {
		d.Title = strings.TrimSpace(m[1])
	}

	// Extract slug
	if m := regexp.MustCompile(`(?m)^Slug:\s*(.+)$`).FindStringSubmatch(content); len(m) > 1 {
		d.Slug = strings.TrimSpace(m[1])
	}

	// Extract widget slugs from list items after ## Widgets
	widgetSection := regexp.MustCompile(`(?ms)^## Widgets\n(.+?)(?:^##|\z)`).FindStringSubmatch(content)
	if len(widgetSection) > 1 {
		for _, m := range regexp.MustCompile(`(?m)^- (.+)$`).FindAllStringSubmatch(widgetSection[1], -1) {
			if len(m) > 1 {
				d.WidgetSlugs = append(d.WidgetSlugs, strings.TrimSpace(m[1]))
			}
		}
	}

	if len(d.WidgetSlugs) == 0 {
		return nil, fmt.Errorf("dashboard markdown has no widget references")
	}

	return d, nil
}

// writeDashboardMarkdown creates a dashboard markdown file on disk.
func writeDashboardMarkdown(dir string, d *ParsedDashboard) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", d.Title))
	sb.WriteString(fmt.Sprintf("Slug: %s\n\n", d.Slug))
	sb.WriteString("## Widgets\n")
	for _, slug := range d.WidgetSlugs {
		sb.WriteString(fmt.Sprintf("- %s\n", slug))
	}

	path := filepath.Join(dir, d.Slug+".dashboard.md")
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

// loadDashboard reads and parses a dashboard markdown file by slug.
func loadDashboard(dir, slug string) (*ParsedDashboard, error) {
	path := filepath.Join(dir, slug+".dashboard.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dashboard %q: %w", slug, err)
	}
	return parseDashboardMarkdown(string(data))
}

// renderDashboardHTML generates a self-contained HTML page with iframes for each widget.
func renderDashboardHTML(d *ParsedDashboard) string {
	var iframes strings.Builder
	for _, slug := range d.WidgetSlugs {
		iframes.WriteString(fmt.Sprintf(`
        <div class="widget-frame">
            <div class="widget-frame-header">%s</div>
            <iframe sandbox="allow-scripts" src="/api/widgets/%s" onload="resizeIframe(this)"></iframe>
        </div>`, slug, slug))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s — Lighthouse Dashboard</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Segoe UI', system-ui, -apple-system, sans-serif;
            background: #F5F2EC;
            color: #2D2A26;
            padding: 24px;
        }
        h1 {
            font-size: 24px;
            font-weight: 600;
            margin-bottom: 20px;
            color: #2D2A26;
        }
        .back-link {
            display: inline-block;
            margin-bottom: 16px;
            color: #2C5F8A;
            text-decoration: none;
            font-size: 14px;
        }
        .back-link:hover { text-decoration: underline; }
        .dashboard-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(480px, 1fr));
            gap: 20px;
        }
        .widget-frame {
            background: #FFFFFF;
            border: 1px solid #E2DDD4;
            border-radius: 8px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.06);
            overflow: hidden;
        }
        .widget-frame-header {
            padding: 10px 14px;
            font-size: 13px;
            font-weight: 600;
            border-bottom: 1px solid #EDEAE4;
            color: #7A756D;
            text-transform: capitalize;
        }
        .widget-frame iframe {
            width: 100%%;
            border: none;
            min-height: 350px;
        }
    </style>
</head>
<body>
    <a class="back-link" href="/">← Back to Chat</a>
    <h1>%s</h1>
    <div class="dashboard-grid">%s</div>
    <script>
        function resizeIframe(iframe) {
            try {
                var doc = iframe.contentDocument || iframe.contentWindow.document;
                var height = doc.documentElement.scrollHeight;
                iframe.style.height = Math.min(Math.max(height, 200), 700) + 'px';
            } catch(e) {
                iframe.style.height = '350px';
            }
        }
    </script>
</body>
</html>`, d.Title, d.Title, iframes.String())
}
