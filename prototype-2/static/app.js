// State
let sessionId = '';
let isStreaming = false;

// DOM refs
const chatMessages = document.getElementById('chatMessages');
const chatInput = document.getElementById('chatInput');
const sendBtn = document.getElementById('sendBtn');
const widgetArea = document.getElementById('widgetArea');
const widgetPanel = document.getElementById('widgetPanel');

// Load saved widgets on startup
loadSavedWidgets();

// Send message
async function sendMessage(e) {
    e.preventDefault();
    const message = chatInput.value.trim();
    if (!message || isStreaming) return;

    chatInput.value = '';
    appendMessage('user', message);
    setStreaming(true);

    // Create assistant message container
    const assistantMsg = appendMessage('assistant', '');
    const contentDiv = assistantMsg.querySelector('.message-content');

    try {
        const resp = await fetch('/api/chat', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ message, session_id: sessionId }),
        });

        if (!resp.ok) {
            contentDiv.textContent = `Error: ${resp.status} ${resp.statusText}`;
            setStreaming(false);
            return;
        }

        await processSSEStream(resp.body, contentDiv);
    } catch (err) {
        contentDiv.textContent = `Connection error: ${err.message}`;
    }

    setStreaming(false);
}

// Process SSE stream from fetch ReadableStream
async function processSSEStream(body, contentDiv) {
    const reader = body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });

        // Parse SSE events from buffer
        const events = [];
        let idx;
        while ((idx = buffer.indexOf('\n\n')) !== -1) {
            const raw = buffer.slice(0, idx);
            buffer = buffer.slice(idx + 2);

            let eventType = 'message';
            let eventData = '';
            for (const line of raw.split('\n')) {
                if (line.startsWith('event: ')) {
                    eventType = line.slice(7);
                } else if (line.startsWith('data: ')) {
                    eventData = line.slice(6);
                }
            }
            if (eventData) {
                events.push({ type: eventType, data: eventData });
            }
        }

        for (const event of events) {
            handleSSEEvent(event, contentDiv);
        }
    }
}

// Handle individual SSE events
function handleSSEEvent(event, contentDiv) {
    let data;
    try {
        data = JSON.parse(event.data);
    } catch {
        return;
    }

    switch (event.type) {
        case 'session':
            sessionId = data.session_id;
            break;

        case 'thinking':
            // Show a subtle thinking indicator
            if (!contentDiv.querySelector('.thinking-indicator')) {
                const indicator = document.createElement('div');
                indicator.className = 'thinking-indicator';
                indicator.innerHTML = '<span class="loading-dots"><span></span><span></span><span></span></span> Thinking...';
                indicator.style.cssText = 'color: var(--text-secondary); font-size: 12px; margin-bottom: 8px; display: flex; align-items: center; gap: 6px;';
                contentDiv.appendChild(indicator);
            }
            break;

        case 'text': {
            // Remove thinking indicator
            const thinking = contentDiv.querySelector('.thinking-indicator');
            if (thinking) thinking.remove();

            // Append text
            const textNode = document.createElement('div');
            textNode.className = 'message-text';
            textNode.innerHTML = renderMarkdown(data.text);
            contentDiv.appendChild(textNode);
            scrollToBottom();
            break;
        }

        case 'tool_use': {
            const toolDiv = document.createElement('div');
            toolDiv.className = 'tool-use';

            const toolNames = {
                'execute_sql': 'SQL Query',
                'create_widget': 'Create Widget',
                'list_widgets': 'List Widgets',
                'create_dashboard': 'Create Dashboard',
            };
            const toolName = toolNames[data.name] || data.name;

            let inputDisplay = '';
            try {
                const inp = JSON.parse(data.input);
                if (data.name === 'execute_sql') {
                    inputDisplay = inp.query;
                } else if (data.name === 'create_widget') {
                    inputDisplay = `Title: ${inp.title}\nSlug: ${inp.slug}`;
                } else if (data.name === 'create_dashboard') {
                    inputDisplay = `Title: ${inp.title}\nWidgets: ${(inp.widget_slugs || []).join(', ')}`;
                } else {
                    inputDisplay = JSON.stringify(inp, null, 2);
                }
            } catch {
                inputDisplay = data.input;
            }

            toolDiv.innerHTML = `
                <div class="tool-use-header" onclick="this.nextElementSibling.classList.toggle('open')">
                    ⚡ ${toolName}
                    <span style="margin-left: auto; font-size: 10px;">▼</span>
                </div>
                <div class="tool-use-body">${escapeHtml(inputDisplay)}</div>
            `;
            toolDiv.dataset.toolId = data.tool_id;
            contentDiv.appendChild(toolDiv);
            scrollToBottom();
            break;
        }

        case 'tool_result': {
            // Find matching tool_use and add result
            const toolUse = contentDiv.querySelector(`[data-tool-id="${data.tool_id}"]`);
            if (toolUse) {
                const body = toolUse.querySelector('.tool-use-body');
                if (data.is_error) {
                    toolUse.classList.add('tool-result-error');
                }
                // Don't append raw text if body was upgraded to rich SQL display
                if (!body.classList.contains('sql-result-body')) {
                    body.textContent += '\n\n→ ' + data.result;
                }
            }
            break;
        }

        case 'sql_result': {
            // Find the matching tool_use for execute_sql and upgrade its body
            // with syntax-highlighted SQL + data table (replaces the plain text)
            const allToolUses = contentDiv.querySelectorAll('.tool-use');
            let targetToolUse = null;
            for (let i = allToolUses.length - 1; i >= 0; i--) {
                const header = allToolUses[i].querySelector('.tool-use-header');
                if (header && header.textContent.includes('SQL Query')) {
                    targetToolUse = allToolUses[i];
                    break;
                }
            }

            if (targetToolUse) {
                const body = targetToolUse.querySelector('.tool-use-body');
                // Replace plain text body with rich content
                body.innerHTML = '';
                body.className = 'tool-use-body sql-result-body';

                // Syntax-highlighted SQL
                const sqlBlock = document.createElement('div');
                sqlBlock.className = 'sql-result-query';
                sqlBlock.innerHTML = highlightSQL(data.query);
                body.appendChild(sqlBlock);

                // Data table
                if (data.columns && data.rows && data.rows.length > 0) {
                    const table = buildResultTable(data.columns, data.rows);
                    body.appendChild(table);

                    const rowInfo = document.createElement('div');
                    rowInfo.className = 'sql-result-info';
                    rowInfo.textContent = `${data.row_count} row${data.row_count !== 1 ? 's' : ''}`;
                    body.appendChild(rowInfo);
                }
            }
            scrollToBottom();
            break;
        }

        case 'widget':
            addWidget(data.slug, data.title);
            break;

        case 'dashboard': {
            // Show a link to the dashboard
            const dashLink = document.createElement('div');
            dashLink.className = 'dashboard-link';
            dashLink.innerHTML = `<a href="/api/dashboards/${escapeHtml(data.slug)}" target="_blank">📊 Open Dashboard: ${escapeHtml(data.title)}</a>`;
            contentDiv.appendChild(dashLink);
            scrollToBottom();
            break;
        }

        case 'error': {
            const errorDiv = document.createElement('div');
            errorDiv.style.cssText = 'color: var(--danger); padding: 8px; background: var(--danger-bg); border-radius: var(--radius-sm); margin-top: 8px;';
            errorDiv.textContent = `Error: ${data.error}`;
            contentDiv.appendChild(errorDiv);
            scrollToBottom();
            break;
        }

        case 'done': {
            // Remove any remaining thinking indicators
            const thinkingDone = contentDiv.querySelector('.thinking-indicator');
            if (thinkingDone) thinkingDone.remove();
            break;
        }
    }
}

// SQL keyword highlighting
function highlightSQL(sql) {
    const keywords = [
        'SELECT', 'FROM', 'WHERE', 'JOIN', 'LEFT', 'RIGHT', 'INNER', 'OUTER',
        'ON', 'AND', 'OR', 'NOT', 'IN', 'AS', 'ORDER', 'BY', 'GROUP', 'HAVING',
        'LIMIT', 'OFFSET', 'UNION', 'ALL', 'DISTINCT', 'COUNT', 'SUM', 'AVG',
        'MIN', 'MAX', 'CASE', 'WHEN', 'THEN', 'ELSE', 'END', 'IS', 'NULL',
        'LIKE', 'BETWEEN', 'EXISTS', 'WITH', 'DESC', 'ASC', 'INSERT', 'UPDATE',
        'DELETE', 'CREATE', 'TABLE', 'INDEX', 'EXPLAIN',
    ];
    let escaped = escapeHtml(sql);
    // Highlight keywords (word boundaries)
    for (const kw of keywords) {
        const re = new RegExp(`\\b(${kw})\\b`, 'gi');
        escaped = escaped.replace(re, '<span class="sql-keyword">$1</span>');
    }
    // Highlight strings
    escaped = escaped.replace(/&#x27;([^&#]*?)&#x27;/g, "<span class=\"sql-string\">'$1'</span>");
    return escaped;
}

// Build a compact data table from columns and rows
function buildResultTable(columns, rows) {
    const wrapper = document.createElement('div');
    wrapper.className = 'sql-result-table-wrapper';

    const table = document.createElement('table');
    table.className = 'sql-result-table';

    // Header
    const thead = document.createElement('thead');
    const headerRow = document.createElement('tr');
    for (const col of columns) {
        const th = document.createElement('th');
        th.textContent = col;
        headerRow.appendChild(th);
    }
    thead.appendChild(headerRow);
    table.appendChild(thead);

    // Body (limit to 20 rows for inline display)
    const tbody = document.createElement('tbody');
    const displayRows = rows.slice(0, 20);
    for (const row of displayRows) {
        const tr = document.createElement('tr');
        for (const cell of row) {
            const td = document.createElement('td');
            td.textContent = cell === null ? 'NULL' : String(cell);
            // Right-align numbers
            if (typeof cell === 'number') {
                td.style.textAlign = 'right';
                // Format with appropriate precision
                td.textContent = Number.isInteger(cell) ? cell.toString() : cell.toFixed(2);
            }
            tr.appendChild(td);
        }
        tbody.appendChild(tr);
    }
    table.appendChild(tbody);
    wrapper.appendChild(table);

    if (rows.length > 20) {
        const more = document.createElement('div');
        more.className = 'sql-result-info';
        more.textContent = `Showing 20 of ${rows.length} rows`;
        wrapper.appendChild(more);
    }

    return wrapper;
}

// Widget management
function addWidget(slug, title) {
    // Remove placeholder
    const placeholder = widgetArea.querySelector('.widget-placeholder');
    if (placeholder) placeholder.remove();

    // Remove existing widget with same slug (refresh)
    const existing = widgetArea.querySelector(`[data-widget-slug="${slug}"]`);
    if (existing) existing.remove();

    const card = document.createElement('div');
    card.className = 'widget-card';
    card.dataset.widgetSlug = slug;
    card.innerHTML = `
        <div class="widget-card-header">
            ${escapeHtml(title)}
            <button class="widget-close" onclick="removeWidget('${slug}')">&times;</button>
        </div>
        <iframe sandbox="allow-scripts" src="/api/widgets/${slug}"></iframe>
    `;

    // Insert at top
    widgetArea.insertBefore(card, widgetArea.firstChild);

    // Auto-resize iframe after load
    const iframe = card.querySelector('iframe');
    iframe.addEventListener('load', () => {
        try {
            const doc = iframe.contentDocument || iframe.contentWindow.document;
            const height = doc.documentElement.scrollHeight;
            iframe.style.height = Math.min(Math.max(height, 200), 600) + 'px';
        } catch {
            // sandbox may block access
            iframe.style.height = '350px';
        }
    });
}

function removeWidget(slug) {
    const card = widgetArea.querySelector(`[data-widget-slug="${slug}"]`);
    if (card) card.remove();

    // Show placeholder if empty
    if (!widgetArea.querySelector('.widget-card')) {
        widgetArea.innerHTML = `
            <div class="widget-placeholder">
                <p>Widgets will appear here when the AI creates visualizations.</p>
                <p class="hint">Try: "Show me a chart of stock levels by brand"</p>
            </div>
        `;
    }
}

function clearWidgets() {
    widgetArea.innerHTML = `
        <div class="widget-placeholder">
            <p>Widgets will appear here when the AI creates visualizations.</p>
            <p class="hint">Try: "Show me a chart of stock levels by brand"</p>
        </div>
    `;
}

// Load saved widgets from server on page load
async function loadSavedWidgets() {
    try {
        const resp = await fetch('/api/widgets');
        if (!resp.ok) return;
        const widgets = await resp.json();
        if (!widgets || widgets.length === 0) return;

        for (const w of widgets) {
            addWidget(w.slug, w.title);
        }
    } catch {
        // Silently fail — saved widgets are a convenience, not critical
    }
}

// Sidebar navigation helper
function askAbout(question) {
    chatInput.value = question;
    chatInput.focus();
}

// Helpers
function appendMessage(role, text) {
    const div = document.createElement('div');
    div.className = `message ${role}`;
    div.innerHTML = `<div class="message-content">${role === 'user' ? escapeHtml(text) : text}</div>`;
    chatMessages.appendChild(div);
    scrollToBottom();
    return div;
}

function scrollToBottom() {
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

function setStreaming(streaming) {
    isStreaming = streaming;
    sendBtn.disabled = streaming;
    chatInput.disabled = streaming;
    if (!streaming) chatInput.focus();
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Simple markdown rendering (bold, italic, code, lists, headers)
function renderMarkdown(text) {
    // Escape HTML first
    let html = escapeHtml(text);

    // Code blocks
    html = html.replace(/```(\w*)\n([\s\S]*?)```/g, (_, lang, code) => {
        return `<pre><code>${code.trim()}</code></pre>`;
    });

    // Inline code
    html = html.replace(/`([^`]+)`/g, '<code>$1</code>');

    // Bold
    html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');

    // Italic
    html = html.replace(/\*(.+?)\*/g, '<em>$1</em>');

    // Headers
    html = html.replace(/^### (.+)$/gm, '<h4>$1</h4>');
    html = html.replace(/^## (.+)$/gm, '<h3>$1</h3>');
    html = html.replace(/^# (.+)$/gm, '<h2>$1</h2>');

    // Unordered lists
    html = html.replace(/^- (.+)$/gm, '<li>$1</li>');
    html = html.replace(/(<li>.*<\/li>\n?)+/g, '<ul>$&</ul>');

    // Ordered lists
    html = html.replace(/^\d+\. (.+)$/gm, '<li>$1</li>');

    // Paragraphs (double newlines)
    html = html.replace(/\n\n/g, '</p><p>');
    html = '<p>' + html + '</p>';

    // Clean up empty paragraphs
    html = html.replace(/<p><\/p>/g, '');
    html = html.replace(/<p>(<h[234]>)/g, '$1');
    html = html.replace(/(<\/h[234]>)<\/p>/g, '$1');
    html = html.replace(/<p>(<ul>)/g, '$1');
    html = html.replace(/(<\/ul>)<\/p>/g, '$1');
    html = html.replace(/<p>(<pre>)/g, '$1');
    html = html.replace(/(<\/pre>)<\/p>/g, '$1');

    // Single newlines to <br> within paragraphs
    html = html.replace(/\n/g, '<br>');

    return html;
}

// Focus input on load
chatInput.focus();
