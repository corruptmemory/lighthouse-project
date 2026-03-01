// ============================================================
// Chat + Widgets (only active on dashboard page)
// ============================================================

(function() {
    var chatForm = document.getElementById('chatForm');
    if (!chatForm) return; // Not on dashboard page — skip all chat init

    var sessionId = '';
    var isStreaming = false;

    var chatMessages = document.getElementById('chatMessages');
    var chatInput = document.getElementById('chatInput');
    var sendBtn = document.getElementById('sendBtn');
    var widgetArea = document.getElementById('widgetArea');

    // Expose sendMessage globally so the form onsubmit can call it
    window.sendMessage = sendMessage;
    window.clearWidgets = clearWidgets;

    // Load saved widgets on startup
    loadSavedWidgets();

    // Focus input
    chatInput.focus();

    async function sendMessage(e, directMessage) {
        if (e) e.preventDefault();
        var message = directMessage || chatInput.value.trim();
        if (!message || isStreaming) return;

        if (!directMessage) {
            chatInput.value = '';
            appendMessage('user', message);
        }
        setStreaming(true);

        var assistantMsg = appendMessage('assistant', '');
        var contentDiv = assistantMsg.querySelector('.message-content');

        try {
            var resp = await fetch('/api/chat', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ message: message, session_id: sessionId }),
            });

            if (!resp.ok) {
                contentDiv.textContent = 'Error: ' + resp.status + ' ' + resp.statusText;
                setStreaming(false);
                return;
            }

            await processSSEStream(resp.body, contentDiv);
        } catch (err) {
            contentDiv.textContent = 'Connection error: ' + err.message;
        }

        setStreaming(false);
    }

    async function processSSEStream(body, contentDiv) {
        var reader = body.getReader();
        var decoder = new TextDecoder();
        var buffer = '';

        while (true) {
            var result = await reader.read();
            if (result.done) break;

            buffer += decoder.decode(result.value, { stream: true });

            var events = [];
            var idx;
            while ((idx = buffer.indexOf('\n\n')) !== -1) {
                var raw = buffer.slice(0, idx);
                buffer = buffer.slice(idx + 2);

                var eventType = 'message';
                var eventData = '';
                var lines = raw.split('\n');
                for (var i = 0; i < lines.length; i++) {
                    if (lines[i].indexOf('event: ') === 0) {
                        eventType = lines[i].slice(7);
                    } else if (lines[i].indexOf('data: ') === 0) {
                        eventData = lines[i].slice(6);
                    }
                }
                if (eventData) {
                    events.push({ type: eventType, data: eventData });
                }
            }

            for (var j = 0; j < events.length; j++) {
                handleSSEEvent(events[j], contentDiv);
            }
        }
    }

    function handleSSEEvent(event, contentDiv) {
        var data;
        try {
            data = JSON.parse(event.data);
        } catch (e) {
            return;
        }

        switch (event.type) {
            case 'session':
                sessionId = data.session_id;
                break;

            case 'thinking':
                if (!contentDiv.querySelector('.thinking-indicator')) {
                    var indicator = document.createElement('div');
                    indicator.className = 'thinking-indicator';
                    indicator.innerHTML = '<span class="loading-dots"><span></span><span></span><span></span></span> Thinking...';
                    indicator.style.cssText = 'color: var(--text-secondary); font-size: 12px; margin-bottom: 8px; display: flex; align-items: center; gap: 6px;';
                    contentDiv.appendChild(indicator);
                }
                break;

            case 'text': {
                var thinking = contentDiv.querySelector('.thinking-indicator');
                if (thinking) thinking.remove();

                var textNode = document.createElement('div');
                textNode.className = 'message-text';
                textNode.innerHTML = renderMarkdown(data.text);
                contentDiv.appendChild(textNode);
                scrollToBottom();
                break;
            }

            case 'tool_use': {
                var toolDiv = document.createElement('div');
                toolDiv.className = 'tool-use';

                var toolNames = {
                    'execute_sql': 'SQL Query',
                    'create_widget': 'Create Widget',
                    'list_widgets': 'List Widgets',
                    'create_dashboard': 'Create Dashboard',
                };
                var toolName = toolNames[data.name] || data.name;

                var inputDisplay = '';
                try {
                    var inp = JSON.parse(data.input);
                    if (data.name === 'execute_sql') {
                        inputDisplay = inp.query;
                    } else if (data.name === 'create_widget') {
                        inputDisplay = 'Title: ' + inp.title + '\nSlug: ' + inp.slug;
                    } else if (data.name === 'create_dashboard') {
                        inputDisplay = 'Title: ' + inp.title + '\nWidgets: ' + (inp.widget_slugs || []).join(', ');
                    } else {
                        inputDisplay = JSON.stringify(inp, null, 2);
                    }
                } catch (e) {
                    inputDisplay = data.input;
                }

                toolDiv.innerHTML =
                    '<div class="tool-use-header" onclick="this.nextElementSibling.classList.toggle(\'open\')">' +
                    '&#9889; ' + toolName +
                    '<span style="margin-left: auto; font-size: 10px;">&#9660;</span>' +
                    '</div>' +
                    '<div class="tool-use-body">' + escapeHtml(inputDisplay) + '</div>';
                toolDiv.dataset.toolId = data.tool_id;
                contentDiv.appendChild(toolDiv);
                scrollToBottom();
                break;
            }

            case 'tool_result': {
                var toolUse = contentDiv.querySelector('[data-tool-id="' + data.tool_id + '"]');
                if (toolUse) {
                    var body = toolUse.querySelector('.tool-use-body');
                    if (data.is_error) {
                        toolUse.classList.add('tool-result-error');
                    }
                    if (!body.classList.contains('sql-result-body')) {
                        body.textContent += '\n\n→ ' + data.result;
                    }
                }
                break;
            }

            case 'sql_result': {
                var allToolUses = contentDiv.querySelectorAll('.tool-use');
                var targetToolUse = null;
                for (var k = allToolUses.length - 1; k >= 0; k--) {
                    var header = allToolUses[k].querySelector('.tool-use-header');
                    if (header && header.textContent.indexOf('SQL Query') !== -1) {
                        targetToolUse = allToolUses[k];
                        break;
                    }
                }

                if (targetToolUse) {
                    var sqlBody = targetToolUse.querySelector('.tool-use-body');
                    sqlBody.innerHTML = '';
                    sqlBody.className = 'tool-use-body sql-result-body';

                    var sqlBlock = document.createElement('div');
                    sqlBlock.className = 'sql-result-query';
                    sqlBlock.innerHTML = highlightSQL(data.query);
                    sqlBody.appendChild(sqlBlock);

                    if (data.columns && data.rows && data.rows.length > 0) {
                        var table = buildResultTable(data.columns, data.rows);
                        sqlBody.appendChild(table);

                        var rowInfo = document.createElement('div');
                        rowInfo.className = 'sql-result-info';
                        rowInfo.textContent = data.row_count + ' row' + (data.row_count !== 1 ? 's' : '');
                        sqlBody.appendChild(rowInfo);

                        var actions = document.createElement('div');
                        actions.className = 'sql-result-actions';
                        var widgetBtn = document.createElement('button');
                        widgetBtn.className = 'make-widget-btn';
                        widgetBtn.textContent = 'Make a widget from this';
                        (function(q, b) {
                            widgetBtn.addEventListener('click', function() { requestWidgetFromSQL(q, b); });
                        })(data.query, widgetBtn);
                        actions.appendChild(widgetBtn);
                        sqlBody.appendChild(actions);
                    }
                }
                scrollToBottom();
                break;
            }

            case 'widget':
                addWidget(data.slug, data.title);
                break;

            case 'dashboard': {
                var dashLink = document.createElement('div');
                dashLink.className = 'dashboard-link';
                dashLink.innerHTML = '<a href="/api/dashboards/' + escapeHtml(data.slug) + '" target="_blank">&#128202; Open Dashboard: ' + escapeHtml(data.title) + '</a>';
                contentDiv.appendChild(dashLink);
                scrollToBottom();
                break;
            }

            case 'error': {
                var errorDiv = document.createElement('div');
                errorDiv.style.cssText = 'color: var(--danger); padding: 8px; background: var(--danger-bg); border-radius: var(--radius-sm); margin-top: 8px;';
                errorDiv.textContent = 'Error: ' + data.error;
                contentDiv.appendChild(errorDiv);
                scrollToBottom();
                break;
            }

            case 'done': {
                var thinkingDone = contentDiv.querySelector('.thinking-indicator');
                if (thinkingDone) thinkingDone.remove();
                break;
            }
        }
    }

    function highlightSQL(sql) {
        var keywords = [
            'SELECT', 'FROM', 'WHERE', 'JOIN', 'LEFT', 'RIGHT', 'INNER', 'OUTER',
            'ON', 'AND', 'OR', 'NOT', 'IN', 'AS', 'ORDER', 'BY', 'GROUP', 'HAVING',
            'LIMIT', 'OFFSET', 'UNION', 'ALL', 'DISTINCT', 'COUNT', 'SUM', 'AVG',
            'MIN', 'MAX', 'CASE', 'WHEN', 'THEN', 'ELSE', 'END', 'IS', 'NULL',
            'LIKE', 'BETWEEN', 'EXISTS', 'WITH', 'DESC', 'ASC', 'INSERT', 'UPDATE',
            'DELETE', 'CREATE', 'TABLE', 'INDEX', 'EXPLAIN',
        ];
        var escaped = escapeHtml(sql);
        for (var i = 0; i < keywords.length; i++) {
            var re = new RegExp('\\b(' + keywords[i] + ')\\b', 'gi');
            escaped = escaped.replace(re, '<span class="sql-keyword">$1</span>');
        }
        escaped = escaped.replace(/&#x27;([^&#]*?)&#x27;/g, "<span class=\"sql-string\">'$1'</span>");
        return escaped;
    }

    function buildResultTable(columns, rows) {
        var wrapper = document.createElement('div');
        wrapper.className = 'sql-result-table-wrapper';

        var table = document.createElement('table');
        table.className = 'sql-result-table';

        var thead = document.createElement('thead');
        var headerRow = document.createElement('tr');
        for (var i = 0; i < columns.length; i++) {
            var th = document.createElement('th');
            th.textContent = columns[i];
            headerRow.appendChild(th);
        }
        thead.appendChild(headerRow);
        table.appendChild(thead);

        var tbody = document.createElement('tbody');
        var displayRows = rows.slice(0, 20);
        for (var r = 0; r < displayRows.length; r++) {
            var tr = document.createElement('tr');
            for (var c = 0; c < displayRows[r].length; c++) {
                var td = document.createElement('td');
                var cell = displayRows[r][c];
                td.textContent = cell === null ? 'NULL' : String(cell);
                if (typeof cell === 'number') {
                    td.style.textAlign = 'right';
                    td.textContent = Number.isInteger(cell) ? cell.toString() : cell.toFixed(2);
                }
                tr.appendChild(td);
            }
            tbody.appendChild(tr);
        }
        table.appendChild(tbody);
        wrapper.appendChild(table);

        if (rows.length > 20) {
            var more = document.createElement('div');
            more.className = 'sql-result-info';
            more.textContent = 'Showing 20 of ' + rows.length + ' rows';
            wrapper.appendChild(more);
        }

        return wrapper;
    }

    function addWidget(slug, title) {
        var placeholder = widgetArea.querySelector('.widget-placeholder');
        if (placeholder) placeholder.remove();

        var existing = widgetArea.querySelector('[data-widget-slug="' + slug + '"]');
        if (existing) existing.remove();

        var card = document.createElement('div');
        card.className = 'widget-card';
        card.dataset.widgetSlug = slug;
        card.innerHTML =
            '<div class="widget-card-header">' +
            escapeHtml(title) +
            '<button class="widget-close" onclick="removeWidget(\'' + slug + '\')">&times;</button>' +
            '</div>' +
            '<iframe sandbox="allow-scripts" src="/api/widgets/' + slug + '"></iframe>';

        widgetArea.insertBefore(card, widgetArea.firstChild);

        var iframe = card.querySelector('iframe');
        iframe.addEventListener('load', function() {
            try {
                var doc = iframe.contentDocument || iframe.contentWindow.document;
                var height = doc.documentElement.scrollHeight;
                iframe.style.height = Math.min(Math.max(height, 200), 600) + 'px';
            } catch (e) {
                iframe.style.height = '350px';
            }
        });
    }

    window.removeWidget = function(slug) {
        var card = widgetArea.querySelector('[data-widget-slug="' + slug + '"]');
        if (card) card.remove();

        if (!widgetArea.querySelector('.widget-card')) {
            widgetArea.innerHTML =
                '<div class="widget-placeholder">' +
                '<p>Widgets will appear here when you create them.</p>' +
                '<p class="hint">Click "Make a widget from this" on any SQL result, or ask directly in chat.</p>' +
                '</div>';
        }
    };

    function clearWidgets() {
        widgetArea.innerHTML =
            '<div class="widget-placeholder">' +
            '<p>Widgets will appear here when you create them.</p>' +
            '<p class="hint">Click "Make a widget from this" on any SQL result, or ask directly in chat.</p>' +
            '</div>';
    }

    function requestWidgetFromSQL(query, btn) {
        if (isStreaming) return;
        btn.disabled = true;
        btn.textContent = 'Creating widget...';
        var message = 'Create a widget visualization for this SQL query:\n```sql\n' + query + '\n```';
        sendMessage(null, message);
    }

    async function loadSavedWidgets() {
        try {
            var resp = await fetch('/api/widgets');
            if (!resp.ok) return;
            var widgets = await resp.json();
            if (!widgets || widgets.length === 0) return;
            for (var i = 0; i < widgets.length; i++) {
                addWidget(widgets[i].slug, widgets[i].title);
            }
        } catch (e) {
            // Silently fail
        }
    }

    function appendMessage(role, text) {
        var div = document.createElement('div');
        div.className = 'message ' + role;
        div.innerHTML = '<div class="message-content">' + (role === 'user' ? escapeHtml(text) : text) + '</div>';
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

    function renderMarkdown(text) {
        var html = escapeHtml(text);
        html = html.replace(/```(\w*)\n([\s\S]*?)```/g, function(_, lang, code) {
            return '<pre><code>' + code.trim() + '</code></pre>';
        });
        html = html.replace(/`([^`]+)`/g, '<code>$1</code>');
        html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
        html = html.replace(/\*(.+?)\*/g, '<em>$1</em>');
        html = html.replace(/^### (.+)$/gm, '<h4>$1</h4>');
        html = html.replace(/^## (.+)$/gm, '<h3>$1</h3>');
        html = html.replace(/^# (.+)$/gm, '<h2>$1</h2>');
        html = html.replace(/^- (.+)$/gm, '<li>$1</li>');
        html = html.replace(/(<li>.*<\/li>\n?)+/g, '<ul>$&</ul>');
        html = html.replace(/^\d+\. (.+)$/gm, '<li>$1</li>');
        html = html.replace(/\n\n/g, '</p><p>');
        html = '<p>' + html + '</p>';
        html = html.replace(/<p><\/p>/g, '');
        html = html.replace(/<p>(<h[234]>)/g, '$1');
        html = html.replace(/(<\/h[234]>)<\/p>/g, '$1');
        html = html.replace(/<p>(<ul>)/g, '$1');
        html = html.replace(/(<\/ul>)<\/p>/g, '$1');
        html = html.replace(/<p>(<pre>)/g, '$1');
        html = html.replace(/(<\/pre>)<\/p>/g, '$1');
        html = html.replace(/\n/g, '<br>');
        return html;
    }
})();


// ============================================================
// Product Autocomplete (active on orders page)
// ============================================================

(function() {
    // Global line item index counter — survives across addLineItem calls
    window._lineIndex = 0;

    // Move line-item management functions to global scope
    window.addLineItem = function() {
        var tbody = document.getElementById('line-items-body');
        if (!tbody) return;
        var idx = window._lineIndex;

        var tr = document.createElement('tr');
        tr.className = 'line-item-row';
        tr.dataset.index = idx;

        tr.innerHTML =
            '<td>' +
                '<input type="hidden" name="items[' + idx + '].product_id" class="line-product-id">' +
                '<input type="text" class="line-product-search" placeholder="Search products..." autocomplete="off">' +
                '<div class="autocomplete-dropdown"></div>' +
            '</td>' +
            '<td class="col-num">' +
                '<input type="number" name="items[' + idx + '].quantity" class="line-qty" value="1" min="1" required onchange="updateLineTotal(this)">' +
            '</td>' +
            '<td class="col-num line-price">$0.00</td>' +
            '<td class="col-num line-total">$0.00</td>' +
            '<td><button type="button" class="btn-remove" onclick="removeLineItem(this)">&times;</button></td>';

        tbody.appendChild(tr);
        window._lineIndex++;

        var searchInput = tr.querySelector('.line-product-search');
        initProductAutocomplete(searchInput);
    };

    window.removeLineItem = function(btn) {
        btn.closest('tr').remove();
        recalcTotal();
    };

    window.updateLineTotal = function(input) {
        var row = input.closest('tr');
        var priceText = row.querySelector('.line-price').textContent;
        var price = parseFloat(priceText.replace('$', '')) || 0;
        var qty = parseInt(input.value) || 0;
        row.querySelector('.line-total').textContent = '$' + (price * qty).toFixed(2);
        recalcTotal();
    };

    // Also keep the old updateLinePrice for existing <select> rows during edit
    window.updateLinePrice = function(sel) {
        var row = sel.closest('tr');
        var opt = sel.options[sel.selectedIndex];
        var price = parseFloat(opt.dataset.price || 0);
        row.querySelector('.line-price').textContent = '$' + price.toFixed(2);
        updateLineTotal(row.querySelector('.line-qty'));
    };

    function recalcTotal() {
        var total = 0;
        document.querySelectorAll('.line-total').forEach(function(td) {
            total += parseFloat(td.textContent.replace('$', '')) || 0;
        });
        var el = document.getElementById('order-total');
        if (el) el.textContent = '$' + total.toFixed(2);
    }
    window.recalcTotal = recalcTotal;

    function initProductAutocomplete(input) {
        var row = input.closest('tr');
        var hiddenInput = row.querySelector('.line-product-id');
        var dropdown = row.querySelector('.autocomplete-dropdown');
        var debounceTimer = null;
        var selectedIdx = -1;
        var currentResults = [];

        input.addEventListener('input', function() {
            var query = input.value.trim();
            clearTimeout(debounceTimer);
            if (query.length < 1) {
                closeDropdown();
                return;
            }
            debounceTimer = setTimeout(function() {
                fetchProducts(query);
            }, 200);
        });

        input.addEventListener('keydown', function(e) {
            if (!dropdown.classList.contains('open')) return;

            if (e.key === 'ArrowDown') {
                e.preventDefault();
                selectedIdx = Math.min(selectedIdx + 1, currentResults.length - 1);
                highlightItem();
            } else if (e.key === 'ArrowUp') {
                e.preventDefault();
                selectedIdx = Math.max(selectedIdx - 1, 0);
                highlightItem();
            } else if (e.key === 'Enter') {
                e.preventDefault();
                if (selectedIdx >= 0 && selectedIdx < currentResults.length) {
                    selectProduct(currentResults[selectedIdx]);
                }
            } else if (e.key === 'Escape') {
                closeDropdown();
            }
        });

        // Close dropdown when clicking outside
        document.addEventListener('click', function(e) {
            if (!input.contains(e.target) && !dropdown.contains(e.target)) {
                closeDropdown();
            }
        });

        async function fetchProducts(query) {
            try {
                var resp = await fetch('/api/products?search=' + encodeURIComponent(query));
                if (!resp.ok) return;
                currentResults = await resp.json();
                renderDropdown();
            } catch (e) {
                // ignore
            }
        }

        function renderDropdown() {
            if (currentResults.length === 0) {
                dropdown.innerHTML = '<div class="autocomplete-empty">No products found</div>';
                dropdown.classList.add('open');
                selectedIdx = -1;
                return;
            }

            dropdown.innerHTML = '';
            selectedIdx = -1;
            for (var i = 0; i < currentResults.length; i++) {
                var item = document.createElement('div');
                item.className = 'autocomplete-item';
                item.dataset.idx = i;
                item.innerHTML =
                    '<span class="ac-name">' + escapeHtml(currentResults[i].name) + '</span>' +
                    '<span class="ac-meta">' + escapeHtml(currentResults[i].sku) + ' &middot; $' + currentResults[i].price.toFixed(2) + '</span>';

                (function(idx) {
                    item.addEventListener('click', function() {
                        selectProduct(currentResults[idx]);
                    });
                })(i);

                dropdown.appendChild(item);
            }
            dropdown.classList.add('open');
        }

        function highlightItem() {
            var items = dropdown.querySelectorAll('.autocomplete-item');
            items.forEach(function(el, i) {
                el.classList.toggle('highlighted', i === selectedIdx);
            });
            // Scroll into view
            if (items[selectedIdx]) {
                items[selectedIdx].scrollIntoView({ block: 'nearest' });
            }
        }

        function selectProduct(product) {
            hiddenInput.value = product.id;
            input.value = product.name + ' (' + product.sku + ')';
            row.querySelector('.line-price').textContent = '$' + product.price.toFixed(2);
            // Recalc line total
            var qty = parseInt(row.querySelector('.line-qty').value) || 1;
            row.querySelector('.line-total').textContent = '$' + (product.price * qty).toFixed(2);
            recalcTotal();
            closeDropdown();
        }

        function closeDropdown() {
            dropdown.innerHTML = '';
            dropdown.classList.remove('open');
            currentResults = [];
            selectedIdx = -1;
        }
    }

    // Make initProductAutocomplete available for dynamic forms loaded via htmx
    window.initProductAutocomplete = initProductAutocomplete;

    // Initialize autocomplete on any existing search inputs (e.g., after htmx loads a form)
    document.addEventListener('htmx:afterSettle', function() {
        document.querySelectorAll('.line-product-search').forEach(function(input) {
            if (!input.dataset.acInit) {
                input.dataset.acInit = '1';
                initProductAutocomplete(input);
            }
        });

        // Update _lineIndex to account for existing rows
        var rows = document.querySelectorAll('.line-item-row');
        if (rows.length > 0) {
            var maxIdx = 0;
            rows.forEach(function(r) {
                var idx = parseInt(r.dataset.index || 0);
                if (idx >= maxIdx) maxIdx = idx + 1;
            });
            window._lineIndex = maxIdx;
        }
    });
})();


// ============================================================
// AI Filter (Orders page)
// ============================================================

function toggleFilterMode(mode) {
    var fields = document.getElementById('filters');
    var ai = document.getElementById('ai-filters');
    if (!fields || !ai) return;

    if (mode === 'ai') {
        fields.style.display = 'none';
        ai.style.display = 'flex';
        document.getElementById('ai-filter-input').focus();
    } else {
        fields.style.display = 'flex';
        ai.style.display = 'none';
        clearAIFilter();
    }
}

function submitAIFilter() {
    var input = document.getElementById('ai-filter-input');
    var btn = document.getElementById('ai-filter-btn');
    if (!input || !btn) return;

    var prompt = input.value.trim();
    if (!prompt) return;

    btn.disabled = true;
    btn.textContent = 'Searching...';
    btn.classList.add('ai-filter-loading');

    fetch('/api/orders/ai-filter', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ prompt: prompt }),
    })
    .then(function(resp) {
        // Extract explanation from HX-Trigger header
        var trigger = resp.headers.get('HX-Trigger');
        if (trigger) {
            try {
                var triggerData = JSON.parse(trigger);
                if (triggerData.aiExplanation) {
                    var banner = document.getElementById('ai-explanation');
                    var text = document.getElementById('ai-explanation-text');
                    if (banner && text) {
                        text.textContent = triggerData.aiExplanation;
                        banner.style.display = 'flex';
                    }
                }
            } catch (e) { /* ignore */ }
        }

        if (!resp.ok) {
            return resp.text().then(function(t) { throw new Error(t); });
        }
        return resp.text();
    })
    .then(function(html) {
        var tbody = document.getElementById('order-table-body');
        if (tbody) tbody.innerHTML = html;
    })
    .catch(function(err) {
        var tbody = document.getElementById('order-table-body');
        if (tbody) {
            tbody.innerHTML = '<tr><td colspan="7" class="empty-state">AI filter error: ' +
                escapeHtml(err.message) + '</td></tr>';
        }
    })
    .finally(function() {
        btn.disabled = false;
        btn.textContent = 'Search';
        btn.classList.remove('ai-filter-loading');
    });
}

function clearAIFilter() {
    var banner = document.getElementById('ai-explanation');
    if (banner) banner.style.display = 'none';

    // Use the existing refreshTable trigger on the table body
    htmx.trigger(document.body, 'refreshTable');
}

// Enter key support for AI filter input
document.addEventListener('keydown', function(e) {
    if (e.key === 'Enter' && e.target.id === 'ai-filter-input') {
        e.preventDefault();
        submitAIFilter();
    }
});

// ============================================================
// AI Bulk Operations Modal (Orders page)
// ============================================================

(function() {
    var modal = document.getElementById('bulk-ops-modal');
    if (!modal) return; // Not on orders page

    var proposedActions = [];
    var didExecute = false;

    window.openBulkOpsModal = function() {
        modal.style.display = 'flex';
        proposedActions = [];
        didExecute = false;

        var content = document.getElementById('bulk-ops-content');
        content.innerHTML =
            '<div class="bulk-ops-placeholder">' +
            'Describe what changes you want to make and click Generate. ' +
            'The AI will propose specific actions for your review.' +
            '</div>';

        var footer = document.getElementById('bulk-ops-footer');
        footer.style.display = 'none';

        var input = document.getElementById('bulk-ops-input');
        input.value = '';
        input.disabled = false;
        var genBtn = document.getElementById('bulk-ops-generate-btn');
        genBtn.disabled = false;
        genBtn.textContent = 'Generate';

        setTimeout(function() { input.focus(); }, 50);
    };

    window.closeBulkOpsModal = function() {
        modal.style.display = 'none';
        if (didExecute) {
            htmx.trigger(document.body, 'refreshTable');
        }
    };

    window.generateBulkOps = function() {
        var input = document.getElementById('bulk-ops-input');
        var btn = document.getElementById('bulk-ops-generate-btn');
        var content = document.getElementById('bulk-ops-content');
        var footer = document.getElementById('bulk-ops-footer');

        var prompt = input.value.trim();
        if (!prompt) return;

        input.disabled = true;
        btn.disabled = true;
        btn.textContent = 'Generating...';
        footer.style.display = 'none';

        content.innerHTML =
            '<div class="bulk-ops-loading">' +
            '<span class="loading-dots"><span></span><span></span><span></span></span> ' +
            'AI is analyzing the database and proposing actions...' +
            '</div>';

        fetch('/api/orders/ai-bulk-ops', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ prompt: prompt }),
        })
        .then(function(resp) {
            if (!resp.ok) {
                return resp.text().then(function(t) { throw new Error(t); });
            }
            return resp.json();
        })
        .then(function(data) {
            proposedActions = data.actions || [];
            renderProposedActions(content, data.summary, proposedActions);
            footer.style.display = 'flex';
            updateBulkOpsCount();
        })
        .catch(function(err) {
            content.innerHTML =
                '<div class="bulk-ops-placeholder" style="color: var(--danger);">' +
                'Error: ' + escapeHtml(err.message) +
                '</div>';
        })
        .finally(function() {
            input.disabled = false;
            btn.disabled = false;
            btn.textContent = 'Generate';
        });
    };

    function renderProposedActions(container, summary, actions) {
        var html = '';

        if (summary) {
            html += '<div class="bulk-ops-summary">' + escapeHtml(summary) + '</div>';
        }

        if (actions.length === 0) {
            html += '<div class="bulk-ops-placeholder">No actions proposed.</div>';
            container.innerHTML = html;
            return;
        }

        html += '<table class="bulk-ops-table">';
        html += '<thead><tr>';
        html += '<th><input type="checkbox" checked onchange="toggleAllBulkOps(this.checked)"></th>';
        html += '<th>Action Description</th>';
        html += '</tr></thead>';
        html += '<tbody>';

        for (var i = 0; i < actions.length; i++) {
            html += '<tr>';
            html += '<td><input type="checkbox" class="bulk-ops-checkbox" data-index="' + i + '" checked onchange="updateBulkOpsCount()"></td>';
            html += '<td>';
            html += '<div class="bulk-ops-description">' + escapeHtml(actions[i].description) + '</div>';
            html += '<div class="bulk-ops-sql">' + escapeHtml(actions[i].sql) + '</div>';
            html += '</td>';
            html += '</tr>';
        }

        html += '</tbody></table>';
        container.innerHTML = html;
    }

    window.toggleAllBulkOps = function(checked) {
        var boxes = document.querySelectorAll('.bulk-ops-checkbox');
        for (var i = 0; i < boxes.length; i++) {
            boxes[i].checked = checked;
        }
        updateBulkOpsCount();
    };

    window.updateBulkOpsCount = function() {
        var boxes = document.querySelectorAll('.bulk-ops-checkbox');
        var count = 0;
        for (var i = 0; i < boxes.length; i++) {
            if (boxes[i].checked) count++;
        }

        var countEl = document.getElementById('bulk-ops-count');
        countEl.textContent = count + ' action' + (count !== 1 ? 's' : '') + ' selected';

        var execBtn = document.getElementById('bulk-ops-execute-btn');
        execBtn.textContent = 'Execute (' + count + ')';
        execBtn.disabled = count === 0;
    };

    window.executeBulkOps = function() {
        var boxes = document.querySelectorAll('.bulk-ops-checkbox');
        var selected = [];
        for (var i = 0; i < boxes.length; i++) {
            if (boxes[i].checked) {
                var idx = parseInt(boxes[i].dataset.index);
                selected.push(proposedActions[idx]);
            }
        }

        if (selected.length === 0) return;

        var execBtn = document.getElementById('bulk-ops-execute-btn');
        execBtn.disabled = true;
        execBtn.textContent = 'Executing...';

        var cancelBtn = document.querySelector('#bulk-ops-footer .btn-secondary');
        cancelBtn.disabled = true;

        fetch('/api/orders/ai-bulk-ops/execute', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ actions: selected }),
        })
        .then(function(resp) {
            if (!resp.ok) {
                return resp.text().then(function(t) { throw new Error(t); });
            }
            return resp.json();
        })
        .then(function(data) {
            didExecute = true;
            renderResults(data);
        })
        .catch(function(err) {
            var content = document.getElementById('bulk-ops-content');
            content.innerHTML +=
                '<div class="bulk-ops-result-message failure">' +
                'Execution error: ' + escapeHtml(err.message) +
                '</div>';
        })
        .finally(function() {
            cancelBtn.disabled = false;
        });
    };

    function renderResults(data) {
        var content = document.getElementById('bulk-ops-content');
        var footer = document.getElementById('bulk-ops-footer');

        var html = '';

        html += '<table class="bulk-ops-table">';
        html += '<thead><tr>';
        html += '<th></th>';
        html += '<th>Action</th>';
        html += '<th style="width: 80px; text-align: right;">Result</th>';
        html += '</tr></thead>';
        html += '<tbody>';

        for (var i = 0; i < data.results.length; i++) {
            var r = data.results[i];
            var cls, icon, meta;

            if (r.success) {
                cls = 'bulk-ops-result-success';
                icon = '&#10003;';
                meta = r.rows_affected + ' row' + (r.rows_affected !== 1 ? 's' : '');
            } else if (r.error && r.error.indexOf('skipped') !== -1) {
                cls = 'bulk-ops-result-skip';
                icon = '&mdash;';
                meta = 'skipped';
            } else {
                cls = 'bulk-ops-result-fail';
                icon = '&#10007;';
                meta = 'Failed';
            }

            html += '<tr>';
            html += '<td class="' + cls + '"><span class="bulk-ops-result-icon">' + icon + '</span></td>';
            html += '<td>';
            html += '<div class="bulk-ops-description">' + escapeHtml(r.description) + '</div>';
            html += '<div class="bulk-ops-sql">' + escapeHtml(r.sql) + '</div>';
            if (r.error && r.error.indexOf('skipped') === -1) {
                html += '<div class="bulk-ops-result-error">' + escapeHtml(r.error) + '</div>';
            }
            html += '</td>';
            html += '<td style="text-align: right;"><span class="bulk-ops-result-meta ' + cls + '">' + meta + '</span></td>';
            html += '</tr>';
        }

        html += '</tbody></table>';

        if (data.success) {
            html += '<div class="bulk-ops-result-message success">' +
                'All actions executed. Transaction committed.' +
                '</div>';
        } else {
            html += '<div class="bulk-ops-result-message failure">' +
                'One or more actions failed. Transaction rolled back — no changes were made.' +
                '</div>';
        }

        content.innerHTML = html;

        // Replace footer with just a Close button
        footer.innerHTML =
            '<span></span>' +
            '<button class="btn btn-primary" onclick="closeBulkOpsModal()">Close</button>';
    }

    // Keyboard shortcuts
    document.addEventListener('keydown', function(e) {
        if (modal.style.display === 'none') return;

        if (e.key === 'Escape') {
            closeBulkOpsModal();
        }

        if (e.key === 'Enter' && e.target.id === 'bulk-ops-input') {
            e.preventDefault();
            generateBulkOps();
        }
    });

    // Click backdrop to close
    modal.addEventListener('click', function(e) {
        if (e.target === modal) {
            closeBulkOpsModal();
        }
    });
})();


// ============================================================
// Shared utilities
// ============================================================

function escapeHtml(text) {
    var div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
