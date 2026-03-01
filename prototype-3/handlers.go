package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// Domain types

type Order struct {
	OrderID         int
	OrderNumber     string
	CustomerName    string
	CustomerEmail   string
	BrandMix        string
	Total           float64
	ItemCount       int
	ShippingCarrier string
	ShippingMethod  string
	Status          string
	OrderDate       string
}

type OrderItem struct {
	ItemID    int
	OrderID   int
	ProductID int
	Quantity  int
	UnitPrice float64
	LineTotal float64
	// Joined fields
	ProductName string
	SKU         string
}

type Product struct {
	ProductID int
	Brand     string
	SKU       string
	Name      string
	Category  string
	Price     float64
}

type Column struct {
	Key   string
	Label string
}

var orderColumns = []Column{
	{"order_number", "Order #"},
	{"customer_name", "Customer"},
	{"brand_mix", "Brand"},
	{"item_count", "Items"},
	{"total", "Total"},
	{"status", "Status"},
	{"order_date", "Date"},
}

// Layout data for the full page render
type LayoutData struct {
	Orders  []Order
	Columns []Column
	SortCol string
	SortDir string
	Page    string // "orders" or "dashboard"
}

// Table partial data
type TableData struct {
	Orders []Order
}

// Detail view data
type DetailData struct {
	Order Order
	Items []OrderItem
}

// Form data
type FormData struct {
	Order    Order
	Items    []OrderItem
	Products []Product
}

// --- Index (full page) ---

func (h *Handlers) HandleIndex(w http.ResponseWriter, r *http.Request) {
	orders, err := h.queryOrders("", "", "", "", "", "order_date", "desc")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	data := LayoutData{
		Orders:  orders,
		Columns: orderColumns,
		SortCol: "order_date",
		SortDir: "desc",
		Page:    "orders",
	}

	if err := h.tmpls.RenderLayout(w, data); err != nil {
		log.Printf("render layout: %v", err)
	}
}

// --- Order list (partial) ---

func (h *Handlers) HandleListOrders(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")
	brand := r.URL.Query().Get("brand")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	sortCol := r.URL.Query().Get("sort")
	sortDir := r.URL.Query().Get("dir")

	if sortCol == "" {
		sortCol = "order_date"
	}
	if sortDir == "" {
		sortDir = "desc"
	}

	orders, err := h.queryOrders(search, status, brand, from, to, sortCol, sortDir)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	data := TableData{Orders: orders}
	if err := h.tmpls.RenderOrderTable(w, data); err != nil {
		log.Printf("render order table: %v", err)
	}
}

// --- Order detail (partial) ---

func (h *Handlers) HandleGetOrder(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid order id", 400)
		return
	}

	order, err := h.getOrder(id)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}

	items, err := h.getOrderItems(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	data := DetailData{Order: order, Items: items}
	if err := h.tmpls.RenderOrderDetail(w, data); err != nil {
		log.Printf("render order detail: %v", err)
	}
}

// --- Edit form (partial) ---

func (h *Handlers) HandleEditOrder(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid order id", 400)
		return
	}

	order, err := h.getOrder(id)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}

	items, err := h.getOrderItems(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	products, err := h.getProducts()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	data := FormData{Order: order, Items: items, Products: products}
	if err := h.tmpls.RenderOrderForm(w, data); err != nil {
		log.Printf("render order form: %v", err)
	}
}

// --- New order form (partial) ---

func (h *Handlers) HandleNewOrder(w http.ResponseWriter, r *http.Request) {
	products, err := h.getProducts()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	data := FormData{
		Order:    Order{Status: "Processing", OrderDate: "2026-02-28"},
		Products: products,
	}
	if err := h.tmpls.RenderOrderForm(w, data); err != nil {
		log.Printf("render order form: %v", err)
	}
}

// --- Create order ---

func (h *Handlers) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", 400)
		return
	}

	// Generate next order number
	orderNum, err := h.nextOrderNumber()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Parse line items
	items := h.parseLineItems(r)
	total := 0.0
	for _, item := range items {
		total += item.LineTotal
	}

	// Determine brand mix
	brandMix := h.determineBrandMix(items)

	result, err := h.db.Exec(`
		INSERT INTO orders (order_number, customer_name, customer_email, brand_mix,
			total, item_count, shipping_carrier, shipping_method, status, order_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		orderNum,
		r.FormValue("customer_name"),
		r.FormValue("customer_email"),
		brandMix,
		total,
		len(items),
		r.FormValue("shipping_carrier"),
		r.FormValue("shipping_method"),
		r.FormValue("status"),
		r.FormValue("order_date"),
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("create order: %v", err), 500)
		return
	}

	orderID, _ := result.LastInsertId()

	// Insert line items
	for _, item := range items {
		_, err := h.db.Exec(`
			INSERT INTO order_items (order_id, product_id, quantity, unit_price, line_total)
			VALUES (?, ?, ?, ?, ?)`,
			orderID, item.ProductID, item.Quantity, item.UnitPrice, item.LineTotal)
		if err != nil {
			log.Printf("insert line item: %v", err)
		}
	}

	// Return detail view of the new order
	order, _ := h.getOrder(int(orderID))
	itemsFull, _ := h.getOrderItems(int(orderID))
	data := DetailData{Order: order, Items: itemsFull}

	// Also trigger a table refresh
	w.Header().Set("HX-Trigger", "refreshTable")
	if err := h.tmpls.RenderOrderDetail(w, data); err != nil {
		log.Printf("render order detail: %v", err)
	}
}

// --- Update order ---

func (h *Handlers) HandleUpdateOrder(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid order id", 400)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", 400)
		return
	}

	// Parse line items
	items := h.parseLineItems(r)
	total := 0.0
	for _, item := range items {
		total += item.LineTotal
	}

	brandMix := h.determineBrandMix(items)

	// If no line items were submitted, keep existing total
	if len(items) == 0 {
		existing, err := h.getOrder(id)
		if err == nil {
			total = existing.Total
			brandMix = existing.BrandMix
		}
	}

	_, err = h.db.Exec(`
		UPDATE orders SET
			customer_name = ?, customer_email = ?, brand_mix = ?,
			total = ?, item_count = ?,
			shipping_carrier = ?, shipping_method = ?,
			status = ?, order_date = ?
		WHERE order_id = ?`,
		r.FormValue("customer_name"),
		r.FormValue("customer_email"),
		brandMix,
		total,
		len(items),
		r.FormValue("shipping_carrier"),
		r.FormValue("shipping_method"),
		r.FormValue("status"),
		r.FormValue("order_date"),
		id,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("update order: %v", err), 500)
		return
	}

	// Replace line items: delete existing, insert new
	h.db.Exec("DELETE FROM order_items WHERE order_id = ?", id)
	for _, item := range items {
		h.db.Exec(`
			INSERT INTO order_items (order_id, product_id, quantity, unit_price, line_total)
			VALUES (?, ?, ?, ?, ?)`,
			id, item.ProductID, item.Quantity, item.UnitPrice, item.LineTotal)
	}

	// Return detail view
	order, _ := h.getOrder(id)
	itemsFull, _ := h.getOrderItems(id)
	data := DetailData{Order: order, Items: itemsFull}

	w.Header().Set("HX-Trigger", "refreshTable")
	if err := h.tmpls.RenderOrderDetail(w, data); err != nil {
		log.Printf("render order detail: %v", err)
	}
}

// --- Product list (for form dropdowns) ---

func (h *Handlers) HandleListProducts(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")

	var products []Product
	var err error
	if search != "" {
		products, err = h.searchProducts(search)
	} else {
		products, err = h.getProducts()
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	// Simple JSON array
	fmt.Fprint(w, "[")
	for i, p := range products {
		if i > 0 {
			fmt.Fprint(w, ",")
		}
		fmt.Fprintf(w, `{"id":%d,"name":%q,"sku":%q,"brand":%q,"price":%.2f}`,
			p.ProductID, p.Name, p.SKU, p.Brand, p.Price)
	}
	fmt.Fprint(w, "]")
}

// --- Database queries ---

func (h *Handlers) queryOrders(search, status, brand, from, to, sortCol, sortDir string) ([]Order, error) {
	query := "SELECT order_id, order_number, customer_name, customer_email, brand_mix, total, item_count, shipping_carrier, shipping_method, status, order_date FROM orders WHERE 1=1"
	var args []any

	if search != "" {
		query += " AND (order_number LIKE ? OR customer_name LIKE ?)"
		pat := "%" + search + "%"
		args = append(args, pat, pat)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if brand != "" {
		query += " AND brand_mix = ?"
		args = append(args, brand)
	}
	if from != "" {
		query += " AND order_date >= ?"
		args = append(args, from)
	}
	if to != "" {
		query += " AND order_date <= ?"
		args = append(args, to)
	}

	// Whitelist sort columns to prevent injection
	validCols := map[string]bool{
		"order_number": true, "customer_name": true, "brand_mix": true,
		"item_count": true, "total": true, "status": true, "order_date": true,
	}
	if !validCols[sortCol] {
		sortCol = "order_date"
	}
	if sortDir != "asc" {
		sortDir = "desc"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", sortCol, sortDir)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.OrderID, &o.OrderNumber, &o.CustomerName, &o.CustomerEmail,
			&o.BrandMix, &o.Total, &o.ItemCount, &o.ShippingCarrier, &o.ShippingMethod,
			&o.Status, &o.OrderDate); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (h *Handlers) getOrder(id int) (Order, error) {
	var o Order
	err := h.db.QueryRow(`
		SELECT order_id, order_number, customer_name, customer_email, brand_mix,
			total, item_count, shipping_carrier, shipping_method, status, order_date
		FROM orders WHERE order_id = ?`, id).Scan(
		&o.OrderID, &o.OrderNumber, &o.CustomerName, &o.CustomerEmail,
		&o.BrandMix, &o.Total, &o.ItemCount, &o.ShippingCarrier, &o.ShippingMethod,
		&o.Status, &o.OrderDate,
	)
	return o, err
}

func (h *Handlers) getOrderItems(orderID int) ([]OrderItem, error) {
	rows, err := h.db.Query(`
		SELECT oi.item_id, oi.order_id, oi.product_id, oi.quantity, oi.unit_price, oi.line_total,
			p.name, p.sku
		FROM order_items oi
		JOIN products p ON oi.product_id = p.product_id
		WHERE oi.order_id = ?
		ORDER BY oi.item_id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []OrderItem
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(&item.ItemID, &item.OrderID, &item.ProductID, &item.Quantity,
			&item.UnitPrice, &item.LineTotal, &item.ProductName, &item.SKU); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (h *Handlers) getProducts() ([]Product, error) {
	rows, err := h.db.Query(`
		SELECT product_id, brand, sku, name, category, price
		FROM products ORDER BY brand, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ProductID, &p.Brand, &p.SKU, &p.Name, &p.Category, &p.Price); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (h *Handlers) nextOrderNumber() (string, error) {
	var maxN int
	err := h.db.QueryRow("SELECT MAX(CAST(SUBSTR(order_number, 4) AS INTEGER)) FROM orders").Scan(&maxN)
	if err == sql.ErrNoRows || maxN == 0 {
		return "LH-5000", nil
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("LH-%d", maxN+1), nil
}

func (h *Handlers) parseLineItems(r *http.Request) []OrderItem {
	var items []OrderItem
	for i := 0; i < 100; i++ {
		pidStr := r.FormValue(fmt.Sprintf("items[%d].product_id", i))
		qtyStr := r.FormValue(fmt.Sprintf("items[%d].quantity", i))
		if pidStr == "" {
			continue
		}
		pid, err := strconv.Atoi(pidStr)
		if err != nil || pid == 0 {
			continue
		}
		qty, err := strconv.Atoi(qtyStr)
		if err != nil || qty < 1 {
			qty = 1
		}

		// Look up product price
		var price float64
		h.db.QueryRow("SELECT price FROM products WHERE product_id = ?", pid).Scan(&price)

		items = append(items, OrderItem{
			ProductID: pid,
			Quantity:  qty,
			UnitPrice: price,
			LineTotal: price * float64(qty),
		})
	}
	return items
}

func (h *Handlers) determineBrandMix(items []OrderItem) string {
	if len(items) == 0 {
		return "Leuchtturm1917"
	}

	brands := make(map[string]bool)
	for _, item := range items {
		var brand string
		h.db.QueryRow("SELECT brand FROM products WHERE product_id = ?", item.ProductID).Scan(&brand)
		if brand != "" {
			brands[brand] = true
		}
	}

	if len(brands) > 1 {
		return "Mixed"
	}
	for b := range brands {
		return b
	}
	return "Leuchtturm1917"
}

func (h *Handlers) searchProducts(query string) ([]Product, error) {
	pat := "%" + query + "%"
	rows, err := h.db.Query(`
		SELECT product_id, brand, sku, name, category, price
		FROM products
		WHERE name LIKE ? OR sku LIKE ? OR brand LIKE ?
		ORDER BY brand, name
		LIMIT 20`, pat, pat, pat)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ProductID, &p.Brand, &p.SKU, &p.Name, &p.Category, &p.Price); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

// --- Dashboard page (full page render) ---

func (h *Handlers) HandleDashboardPage(w http.ResponseWriter, r *http.Request) {
	if err := h.tmpls.RenderDashboard(w); err != nil {
		log.Printf("render dashboard: %v", err)
	}
}

// --- Chat handler (SSE streaming) ---

type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
}

func (h *Handlers) HandleChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	session, created := h.sessions.GetOrCreate(req.SessionID)
	if created {
		log.Printf("New session: %s", session.ID)
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send session ID first
	writeSSE(w, flusher, "session", map[string]string{"session_id": session.ID})

	// Run agent loop with SSE emission
	emit := func(eventType string, data interface{}) {
		writeSSE(w, flusher, eventType, data)
	}

	if err := runAgentLoop(r.Context(), h.agentCfg, session, req.Message, emit); err != nil {
		log.Printf("Agent error: %v", err)
		writeSSE(w, flusher, "error", map[string]string{"error": err.Error()})
	}
}

// --- Widget handlers ---

func (h *Handlers) HandleGetWidgetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	widget, err := loadWidget(h.widgetsDir, slug)
	if err != nil {
		log.Printf("Widget load error (%s): %v", slug, err)
		http.NotFound(w, r)
		return
	}

	html, err := renderWidget(h.db, widget)
	if err != nil {
		log.Printf("Widget render error (%s): %v", slug, err)
		http.Error(w, fmt.Sprintf("Error rendering widget: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (h *Handlers) HandleDashboardView(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	dashboard, err := loadDashboard(h.widgetsDir, slug)
	if err != nil {
		log.Printf("Dashboard load error (%s): %v", slug, err)
		http.NotFound(w, r)
		return
	}

	html := renderDashboardHTML(dashboard)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (h *Handlers) HandleListWidgets(w http.ResponseWriter, r *http.Request) {
	widgets, err := listWidgets(h.widgetsDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error listing widgets: %v", err), http.StatusInternalServerError)
		return
	}

	type widgetInfo struct {
		Slug  string `json:"slug"`
		Title string `json:"title"`
	}

	var result []widgetInfo
	for _, wg := range widgets {
		result = append(result, widgetInfo{Slug: wg.Slug, Title: wg.Title})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("SSE marshal error: %v", err)
		return
	}

	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(jsonData))
	flusher.Flush()
}
