package main

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const schemaDDL = `
CREATE TABLE IF NOT EXISTS products (
    product_id INTEGER PRIMARY KEY AUTOINCREMENT,
    brand TEXT NOT NULL,
    sub_brand TEXT DEFAULT '',
    sku TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    price REAL NOT NULL,
    stock_on_hand INTEGER NOT NULL DEFAULT 0,
    reorder_point INTEGER NOT NULL DEFAULT 10,
    edition_theme TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS orders (
    order_id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_number TEXT NOT NULL UNIQUE,
    customer_name TEXT NOT NULL,
    customer_email TEXT NOT NULL,
    brand_mix TEXT NOT NULL,
    total REAL NOT NULL,
    item_count INTEGER NOT NULL,
    shipping_carrier TEXT NOT NULL,
    shipping_method TEXT NOT NULL,
    status TEXT NOT NULL,
    order_date DATE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS order_items (
    item_id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id INTEGER NOT NULL REFERENCES orders(order_id),
    product_id INTEGER NOT NULL REFERENCES products(product_id),
    quantity INTEGER NOT NULL DEFAULT 1,
    unit_price REAL NOT NULL,
    line_total REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS subscriptions (
    subscription_id INTEGER PRIMARY KEY AUTOINCREMENT,
    subscriber_name TEXT NOT NULL,
    subscriber_email TEXT NOT NULL,
    plan_name TEXT NOT NULL,
    countries TEXT NOT NULL,
    since_year INTEGER NOT NULL,
    next_renewal DATE,
    annual_value REAL NOT NULL,
    status TEXT NOT NULL
);
`

func InitDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schemaDDL); err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}

	if err := seedProducts(db); err != nil {
		return nil, fmt.Errorf("seed products: %w", err)
	}
	if err := seedOrders(db); err != nil {
		return nil, fmt.Errorf("seed orders: %w", err)
	}
	if err := seedSubscriptions(db); err != nil {
		return nil, fmt.Errorf("seed subscriptions: %w", err)
	}

	return db, nil
}

func seedProducts(db *sql.DB) error {
	type product struct {
		brand, subBrand, sku, name, category string
		price                                float64
		stock, reorder                       int
		edition                              string
	}

	products := []product{
		// Leuchtturm1917 — Notebooks
		{"Leuchtturm1917", "", "LT-NB-A5-SUN", "Medium A5 Notebook — Sunflower", "Notebooks", 29.95, 3, 50, "Grow Your Mind"},
		{"Leuchtturm1917", "", "LT-NB-A5-IND", "Medium A5 Notebook — Indigo", "Notebooks", 29.95, 142, 50, "Grow Your Mind"},
		{"Leuchtturm1917", "", "LT-NB-A5-CHR", "Medium A5 Notebook — Cherry", "Notebooks", 29.95, 88, 50, "Grow Your Mind"},
		{"Leuchtturm1917", "", "LT-NB-A5-PMP", "Medium A5 Notebook — Pumpkin", "Notebooks", 29.95, 65, 50, "Grow Your Mind"},
		{"Leuchtturm1917", "", "LT-NB-A6-BLK", "Pocket A6 Notebook — Black", "Notebooks", 19.95, 210, 75, ""},
		{"Leuchtturm1917", "", "LT-NB-B5-NVY", "Composition B5 Notebook — Navy", "Notebooks", 34.95, 47, 30, ""},

		// Leuchtturm1917 — Special Editions
		{"Leuchtturm1917", "", "LT-SE-YOH-A5", "Year of the Horse A5 Gift Set", "Special Editions", 59.95, 28, 15, "Limited"},
		{"Leuchtturm1917", "Bethge", "LT-SE-LUM-A5", "Luminous Notebook A5", "Special Editions", 39.95, 67, 20, "Pop Art"},
		{"Leuchtturm1917", "Monocle", "LT-SE-MON-A5", "Monocle A5 Notebook", "Special Editions", 44.95, 53, 20, "Urban Lifestyle"},

		// Leuchtturm1917 — Writing Instruments
		{"Leuchtturm1917", "", "LT-WI-DG1-BLK", "Drehgriffel Nr. 1 — Black", "Writing Instruments", 34.95, 7, 40, ""},
		{"Leuchtturm1917", "", "LT-WI-DG2-WHT", "Drehgriffel Nr. 2 — White", "Writing Instruments", 29.95, 18, 30, ""},
		{"Leuchtturm1917", "", "LT-WI-DG1-FOX", "Drehgriffel Nr. 1 — Fox Red", "Writing Instruments", 34.95, 35, 40, ""},

		// Leuchtturm1917 — Journals
		{"Leuchtturm1917", "", "LT-JN-BJ-T25", "Bullet Journal — Turquoise25", "Journals", 24.95, 9, 30, ""},
		{"Leuchtturm1917", "", "LT-JN-SLA-BLK", "Some Lines A Day — Black", "Journals", 24.95, 94, 30, ""},

		// Lighthouse Publications — Coin Storage
		{"Lighthouse Publications", "", "LP-CS-QDR-41", "QUADRUM Snaplock 10-Pack (41mm)", "Coin Storage", 24.99, 11, 25, ""},
		{"Lighthouse Publications", "", "LP-CS-QSL-5PK", "QUICKSLAB 5-Pack", "Coin Storage", 34.99, 89, 25, ""},
		{"Lighthouse Publications", "INTERCEPT", "LP-CS-INT-LRG", "INTERCEPT Storage Box (Large)", "Coin Storage", 49.99, 34, 15, ""},
		{"Lighthouse Publications", "", "LP-CS-QDR-26", "QUADRUM Snaplock 10-Pack (26mm)", "Coin Storage", 24.99, 156, 25, ""},

		// Lighthouse Publications — Presentation
		{"Lighthouse Publications", "", "LP-PR-VLT-WD", "VOLTERRA Presentation Case", "Presentation", 89.99, 22, 10, "Wood"},
		{"Lighthouse Publications", "", "LP-PR-NBL-LTH", "NOBILE Presentation Case", "Presentation", 69.99, 31, 10, "Leatherette"},

		// KABE Supplements
		{"Lighthouse Publications", "KABE", "LP-KS-2024-DE", "KABE Supplement 2024 — Germany", "Stamp Supplements", 79.99, 15, 30, "2024"},
		{"Lighthouse Publications", "KABE", "LP-KS-2024-US", "KABE Supplement 2024 — USA", "Stamp Supplements", 69.99, 42, 30, "2024"},
		{"Lighthouse Publications", "KABE", "LP-KS-2024-CH", "KABE Supplement 2024 — Switzerland", "Stamp Supplements", 79.99, 27, 20, "2024"},

		// Lighthouse Publications — Optical
		{"Lighthouse Publications", "", "LP-OPT-HLM-UV", "HOLMES UV Lamp", "Optical", 129.99, 19, 10, ""},
		{"Lighthouse Publications", "", "LP-OPT-L81-DUV", "L81 Dual UV Lamp", "Optical", 89.99, 14, 10, ""},
	}

	stmt, err := db.Prepare(`INSERT INTO products (brand, sub_brand, sku, name, category, price, stock_on_hand, reorder_point, edition_theme) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range products {
		if _, err := stmt.Exec(p.brand, p.subBrand, p.sku, p.name, p.category, p.price, p.stock, p.reorder, p.edition); err != nil {
			return fmt.Errorf("insert %s: %w", p.sku, err)
		}
	}
	return nil
}

func seedOrders(db *sql.DB) error {
	type order struct {
		number, customer, email, brandMix string
		total                             float64
		items                             int
		carrier, method, status, date     string
	}

	orders := []order{
		{"LH-4521", "Sarah Mitchell", "s.mitchell@gmail.com", "Leuchtturm1917", 89.85, 3, "FedEx", "Ground", "Processing", "2026-02-26"},
		{"LH-4520", "Robert Chen", "r.chen@outlook.com", "Lighthouse Publications", 79.99, 1, "USPS", "Priority", "Shipped", "2026-02-26"},
		{"LH-4519", "Emma Schneider", "e.schneider@web.de", "Mixed", 164.93, 5, "UPS", "Ground", "Processing", "2026-02-25"},
		{"LH-4518", "James O'Brien", "j.obrien@yahoo.com", "Leuchtturm1917", 64.90, 2, "USPS", "Flat Rate", "Delivered", "2026-02-25"},
		{"LH-4517", "Maria Gonzalez", "m.gonzalez@icloud.com", "Lighthouse Publications", 214.97, 3, "FedEx", "Ground", "Delivered", "2026-02-25"},
		{"LH-4516", "Thomas Klein", "t.klein@gmx.de", "Leuchtturm1917", 59.95, 1, "UPS", "Ground", "Shipped", "2026-02-24"},
		{"LH-4515", "Alice Park", "a.park@gmail.com", "Leuchtturm1917", 129.80, 4, "FedEx", "Ground", "Delivered", "2026-02-24"},
		{"LH-4514", "Frank Bauer", "f.bauer@posteo.de", "Lighthouse Publications", 159.98, 2, "USPS", "Priority", "Delivered", "2026-02-24"},
		{"LH-4513", "Susan Harper", "s.harper@aol.com", "Leuchtturm1917", 24.95, 1, "USPS", "Flat Rate", "Delivered", "2026-02-23"},
		{"LH-4512", "Markus Weiss", "m.weiss@gmail.com", "Lighthouse Publications", 129.99, 1, "FedEx", "Ground", "Return Requested", "2026-02-23"},
		// Additional orders for richer dataset
		{"LH-4511", "Linda Fox", "l.fox@gmail.com", "Leuchtturm1917", 59.90, 2, "UPS", "Ground", "Delivered", "2026-02-22"},
		{"LH-4510", "Karl Braun", "k.braun@web.de", "Mixed", 94.94, 3, "FedEx", "Ground", "Delivered", "2026-02-22"},
		{"LH-4509", "Jennifer Lee", "j.lee@outlook.com", "Lighthouse Publications", 24.99, 1, "USPS", "Flat Rate", "Delivered", "2026-02-22"},
		{"LH-4508", "Peter Neumann", "p.neumann@gmx.de", "Leuchtturm1917", 44.95, 1, "USPS", "Priority", "Delivered", "2026-02-21"},
		{"LH-4507", "Nancy Wilson", "n.wilson@yahoo.com", "Leuchtturm1917", 89.85, 3, "FedEx", "Ground", "Delivered", "2026-02-21"},
		{"LH-4506", "Michael Torres", "m.torres@gmail.com", "Lighthouse Publications", 159.98, 2, "UPS", "Ground", "Delivered", "2026-02-20"},
		{"LH-4505", "Heike Schmidt", "h.schmidt@posteo.de", "Mixed", 114.94, 4, "FedEx", "Ground", "Delivered", "2026-02-20"},
		{"LH-4504", "Richard Adams", "r.adams@icloud.com", "Leuchtturm1917", 34.95, 1, "USPS", "Flat Rate", "Delivered", "2026-02-19"},
		{"LH-4503", "Claudia Wolf", "c.wolf@web.de", "Lighthouse Publications", 89.99, 1, "FedEx", "Ground", "Delivered", "2026-02-19"},
		{"LH-4502", "David Park", "d.park@gmail.com", "Leuchtturm1917", 54.90, 2, "UPS", "Ground", "Delivered", "2026-02-18"},
		{"LH-4501", "Anna Fischer", "a.fischer@gmx.de", "Mixed", 174.93, 5, "FedEx", "Ground", "Delivered", "2026-02-18"},
		// Older orders for weekly analysis
		{"LH-4500", "William Scott", "w.scott@outlook.com", "Leuchtturm1917", 29.95, 1, "USPS", "Flat Rate", "Delivered", "2026-02-17"},
		{"LH-4499", "Ingrid Muller", "i.muller@gmx.de", "Lighthouse Publications", 159.98, 2, "FedEx", "Ground", "Delivered", "2026-02-17"},
		{"LH-4498", "Chris Taylor", "c.taylor@gmail.com", "Mixed", 109.94, 3, "UPS", "Ground", "Delivered", "2026-02-16"},
		{"LH-4497", "Sabine Lang", "s.lang@posteo.de", "Leuchtturm1917", 59.95, 1, "USPS", "Priority", "Delivered", "2026-02-16"},
		{"LH-4496", "Steven Johnson", "s.johnson@yahoo.com", "Lighthouse Publications", 49.99, 1, "FedEx", "Ground", "Delivered", "2026-02-15"},
		{"LH-4495", "Martina Keller", "m.keller@web.de", "Leuchtturm1917", 74.90, 3, "UPS", "Ground", "Delivered", "2026-02-15"},
		{"LH-4494", "George Harris", "g.harris@icloud.com", "Mixed", 144.94, 4, "FedEx", "Ground", "Delivered", "2026-02-14"},
		{"LH-4493", "Eva Richter", "e.richter@gmx.de", "Leuchtturm1917", 44.95, 1, "USPS", "Flat Rate", "Delivered", "2026-02-14"},
	}

	stmt, err := db.Prepare(`INSERT INTO orders (order_number, customer_name, customer_email, brand_mix, total, item_count, shipping_carrier, shipping_method, status, order_date) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, o := range orders {
		if _, err := stmt.Exec(o.number, o.customer, o.email, o.brandMix, o.total, o.items, o.carrier, o.method, o.status, o.date); err != nil {
			return fmt.Errorf("insert %s: %w", o.number, err)
		}
	}

	// Seed some order_items for the most recent orders
	type orderItem struct {
		orderNum string
		sku      string
		qty      int
		price    float64
	}

	items := []orderItem{
		{"LH-4521", "LT-NB-A5-SUN", 1, 29.95},
		{"LH-4521", "LT-NB-A5-IND", 1, 29.95},
		{"LH-4521", "LT-NB-A5-CHR", 1, 29.95},
		{"LH-4520", "LP-KS-2024-DE", 1, 79.99},
		{"LH-4519", "LT-NB-A5-PMP", 2, 29.95},
		{"LH-4519", "LP-CS-QDR-41", 1, 24.99},
		{"LH-4519", "LP-CS-QSL-5PK", 1, 34.99},
		{"LH-4519", "LT-WI-DG1-BLK", 1, 34.95},
		{"LH-4518", "LT-NB-A6-BLK", 1, 19.95},
		{"LH-4518", "LT-SE-MON-A5", 1, 44.95},
		{"LH-4517", "LP-PR-VLT-WD", 1, 89.99},
		{"LH-4517", "LP-CS-INT-LRG", 1, 49.99},
		{"LH-4517", "LP-CS-QSL-5PK", 1, 34.99},
	}

	for _, item := range items {
		_, err := db.Exec(`
			INSERT INTO order_items (order_id, product_id, quantity, unit_price, line_total)
			SELECT o.order_id, p.product_id, ?, ?, ?
			FROM orders o, products p
			WHERE o.order_number = ? AND p.sku = ?`,
			item.qty, item.price, float64(item.qty)*item.price,
			item.orderNum, item.sku)
		if err != nil {
			return fmt.Errorf("insert item %s/%s: %w", item.orderNum, item.sku, err)
		}
	}

	return nil
}

func seedSubscriptions(db *sql.DB) error {
	type sub struct {
		name, email, plan, countries string
		since                        int
		renewal                      string
		value                        float64
		status                       string
	}

	subs := []sub{
		{"Gerald Hoffman", "g.hoffman@gmail.com", "KABE", "Germany", 2019, "2026-03-01", 79.99, "Renewal Due"},
		{"Patricia Ruiz", "p.ruiz@yahoo.com", "Lighthouse", "USA", 2021, "2026-03-01", 69.99, "Renewal Due"},
		{"David Whitmore", "d.whitmore@icloud.com", "KABE", "Switzerland", 2020, "2026-03-03", 79.99, "Active"},
		{"Sylvia Braun", "s.braun@web.de", "KABE + Lighthouse", "Germany, USA", 2018, "2026-03-05", 149.98, "Active"},
		{"Michael Torres", "m.torres@gmail.com", "Lighthouse", "USA", 2022, "2026-04-12", 69.99, "Active"},
		{"Ingrid Muller", "i.muller@gmx.de", "KABE", "Germany, Austria", 2017, "2026-04-15", 159.98, "Active"},
		{"William Scott", "w.scott@outlook.com", "Lighthouse", "USA, Canada", 2023, "2026-05-01", 139.98, "Active"},
		{"Hans Dietrich", "h.dietrich@posteo.de", "KABE", "Germany", 2024, "", 79.99, "Payment Failed"},
	}

	stmt, err := db.Prepare(`INSERT INTO subscriptions (subscriber_name, subscriber_email, plan_name, countries, since_year, next_renewal, annual_value, status) VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, s := range subs {
		if _, err := stmt.Exec(s.name, s.email, s.plan, s.countries, s.since, s.renewal, s.value, s.status); err != nil {
			return fmt.Errorf("insert sub %s: %w", s.name, err)
		}
	}
	return nil
}
