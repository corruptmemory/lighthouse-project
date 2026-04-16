# KABE Supplement 2024 Orders

Generated: 2026-02-28 23:24
Slug: kabe-supplement-2024-orders

## Context
User requested to see all orders containing KABE Supplement 2024 products. The query finds orders by joining with order_items and products tables to filter by product name.

## SQL
```sql
SELECT DISTINCT 
    o.order_number,
    o.customer_name,
    o.customer_email,
    o.order_date,
    o.total,
    o.status,
    p.name as product_name,
    oi.quantity,
    oi.line_total
FROM orders o
JOIN order_items oi ON o.order_id = oi.order_id
JOIN products p ON oi.product_id = p.product_id
WHERE p.name LIKE '%KABE Supplement 2024%'
ORDER BY o.order_date DESC, o.order_number
```

## Widget
```html
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>KABE Supplement 2024 Orders</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Segoe UI', system-ui, sans-serif;
            background: #F5F2EC;
            padding: 16px;
            color: #2D2A26;
        }
        
        .container {
            background: #FFFFFF;
            border-radius: 8px;
            border: 1px solid #E2DDD4;
            box-shadow: 0 1px 3px rgba(0,0,0,0.06);
            overflow: hidden;
        }
        
        .header {
            background: #2C5F8A;
            color: white;
            padding: 16px 20px;
            font-weight: 600;
            font-size: 18px;
        }
        
        .content {
            padding: 20px;
        }
        
        .order-card {
            border: 1px solid #E2DDD4;
            border-radius: 8px;
            margin-bottom: 16px;
            overflow: hidden;
        }
        
        .order-card:last-child {
            margin-bottom: 0;
        }
        
        .order-header {
            background: #F8F6F3;
            padding: 12px 16px;
            border-bottom: 1px solid #E2DDD4;
            display: grid;
            grid-template-columns: 1fr auto auto;
            gap: 16px;
            align-items: center;
        }
        
        .order-number {
            font-weight: 600;
            color: #2C5F8A;
        }
        
        .order-date {
            color: #7A756D;
            font-size: 14px;
        }
        
        .status-badge {
            padding: 4px 12px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: 500;
            text-transform: uppercase;
        }
        
        .status-processing {
            background: #FDF5E6;
            color: #C4860C;
        }
        
        .status-shipped {
            background: #EDF5EF;
            color: #4A7C59;
        }
        
        .order-details {
            padding: 16px;
        }
        
        .customer-info {
            margin-bottom: 12px;
        }
        
        .customer-name {
            font-weight: 500;
            margin-bottom: 4px;
        }
        
        .customer-email {
            color: #7A756D;
            font-size: 14px;
        }
        
        .product-info {
            background: #F8F6F3;
            padding: 12px;
            border-radius: 6px;
            margin-bottom: 12px;
        }
        
        .product-name {
            font-weight: 500;
            margin-bottom: 4px;
            color: #2C5F8A;
        }
        
        .product-details {
            display: flex;
            justify-content: space-between;
            font-size: 14px;
            color: #7A756D;
        }
        
        .order-total {
            text-align: right;
            font-weight: 600;
            font-size: 18px;
            color: #2C5F8A;
        }
        
        .no-data {
            text-align: center;
            color: #7A756D;
            padding: 40px 20px;
        }
        
        .summary {
            background: #F8F6F3;
            padding: 16px 20px;
            border-top: 1px solid #E2DDD4;
            text-align: center;
            color: #7A756D;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">KABE Supplement 2024 Orders</div>
        <div class="content" id="content">
            <div class="no-data">Loading...</div>
        </div>
        <div class="summary" id="summary"></div>
    </div>

    <script>
        const data = window.WIDGET_DATA;
        const content = document.getElementById('content');
        const summary = document.getElementById('summary');
        
        if (!data || !data.rows || data.rows.length === 0) {
            content.innerHTML = '<div class="no-data">No KABE Supplement 2024 orders found</div>';
            summary.innerHTML = '';
            return;
        }
        
        // Parse column indices
        const colIndex = {};
        data.columns.forEach((col, idx) => {
            colIndex[col] = idx;
        });
        
        let html = '';
        let totalRevenue = 0;
        
        data.rows.forEach(row => {
            const orderNumber = row[colIndex.order_number];
            const customerName = row[colIndex.customer_name];
            const customerEmail = row[colIndex.customer_email];
            const orderDate = new Date(row[colIndex.order_date]).toLocaleDateString();
            const total = parseFloat(row[colIndex.total]);
            const status = row[colIndex.status];
            const productName = row[colIndex.product_name];
            const quantity = row[colIndex.quantity];
            const lineTotal = parseFloat(row[colIndex.line_total]);
            
            totalRevenue += total;
            
            const statusClass = status.toLowerCase().replace(' ', '-');
            
            html += `
                <div class="order-card">
                    <div class="order-header">
                        <div class="order-number">${orderNumber}</div>
                        <div class="order-date">${orderDate}</div>
                        <div class="status-badge status-${statusClass}">${status}</div>
                    </div>
                    <div class="order-details">
                        <div class="customer-info">
                            <div class="customer-name">${customerName}</div>
                            <div class="customer-email">${customerEmail}</div>
                        </div>
                        <div class="product-info">
                            <div class="product-name">${productName}</div>
                            <div class="product-details">
                                <span>Quantity: ${quantity}</span>
                                <span>$${lineTotal.toFixed(2)}</span>
                            </div>
                        </div>
                        <div class="order-total">Total: $${total.toFixed(2)}</div>
                    </div>
                </div>
            `;
        });
        
        content.innerHTML = html;
        summary.innerHTML = `${data.rows.length} orders • Total Revenue: $${totalRevenue.toFixed(2)}`;
    </script>
</body>
</html>
```
