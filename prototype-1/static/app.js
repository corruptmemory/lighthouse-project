// State
let sessionId = '';
let isStreaming = false;

// DOM refs
const chatMessages = document.getElementById('chatMessages');
const chatInput = document.getElementById('chatInput');
const sendBtn = document.getElementById('sendBtn');
const widgetArea = document.getElementById('widgetArea');
const widgetPanel = document.getElementById('widgetPanel');

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
    let currentText = '';

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
            handleSSEEvent(event, contentDiv, (text) => {
                currentText = text;
            });
        }
    }
}

// Handle individual SSE events
function handleSSEEvent(event, contentDiv, setText) {
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

        case 'text':
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

        case 'tool_use': {
            const toolDiv = document.createElement('div');
            toolDiv.className = 'tool-use';

            const toolName = data.name === 'execute_sql' ? 'SQL Query' : 'Render Widget';
            let inputDisplay = '';
            try {
                const inp = JSON.parse(data.input);
                inputDisplay = data.name === 'execute_sql' ? inp.query : `Title: ${inp.title}`;
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
                body.textContent += '\n\n→ ' + data.result;
            }
            break;
        }

        case 'widget':
            addWidget(data.widget_id, data.title);
            break;

        case 'error':
            const errorDiv = document.createElement('div');
            errorDiv.style.cssText = 'color: var(--danger); padding: 8px; background: var(--danger-bg); border-radius: var(--radius-sm); margin-top: 8px;';
            errorDiv.textContent = `Error: ${data.error}`;
            contentDiv.appendChild(errorDiv);
            scrollToBottom();
            break;

        case 'done':
            // Remove any remaining thinking indicators
            const thinkingDone = contentDiv.querySelector('.thinking-indicator');
            if (thinkingDone) thinkingDone.remove();
            break;
    }
}

// Widget management
function addWidget(widgetId, title) {
    // Remove placeholder
    const placeholder = widgetArea.querySelector('.widget-placeholder');
    if (placeholder) placeholder.remove();

    const card = document.createElement('div');
    card.className = 'widget-card';
    card.dataset.widgetId = widgetId;
    card.innerHTML = `
        <div class="widget-card-header">
            ${escapeHtml(title)}
            <button class="widget-close" onclick="removeWidget('${widgetId}')">&times;</button>
        </div>
        <iframe sandbox="allow-scripts" src="/api/widgets/${widgetId}"></iframe>
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

function removeWidget(widgetId) {
    const card = widgetArea.querySelector(`[data-widget-id="${widgetId}"]`);
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
