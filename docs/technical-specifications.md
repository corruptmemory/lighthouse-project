### Technical Specification: Lighthouse Back-End System

#### 1. System Overview and Architectural Vision

The Lighthouse Back-End System is a commerce engine designed to orchestrate the operations of two distinct yet synergistic business units: **Leuchtturm1917** (premium stationery) and **Lighthouse Publications** (numismatic and philatelic collectibles, including the **KABE** brand). The architecture must support a unified core for order management and fulfillment while providing brand-specific data extensions for specialized inventory. This system is designed for multi-region scalability, handling high-volume stationery retail alongside the high-specificity metadata required for professional stamp and coin collecting.

##### Core Objective

A modular, high-performance back-end capable of managing intricate product hierarchies, international localization for diverse global markets (including specific "Other Countries" handlers), and a specialized subscription model for annual collectible updates.

##### Technology Stack

The system is implemented in **Go**, using:

- **Router:** chi (`github.com/go-chi/chi/v5`)
- **Templates:** templ (`github.com/a-h/templ`)
- **Frontend interactivity:** htmx (vendored JS, no npm/node)
- **Database:** PostgreSQL
- **Config:** TOML via `BurntSushi/toml`

All data structures must be strictly typed using Go structs and constants. Business logic must adhere to the brand-specific thresholds and regional mappings defined herein.

#### 2. Entity-Relationship & Data Schema Specifications

The schema utilizes a polymorphic approach to handle generic product data and specialized collectible attributes, including proprietary protection technologies.

##### Products Table (Core)

```sql
CREATE TABLE products (
    product_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand         VARCHAR(50) NOT NULL,   -- 'Leuchtturm1917' or 'Lighthouse Publications'
    sub_brand     VARCHAR(50),            -- 'KABE', 'Monocle', 'Bethge'
    sku           VARCHAR(100) UNIQUE NOT NULL,
    name          VARCHAR(255) NOT NULL,
    category_path TEXT NOT NULL,
    format        VARCHAR(50),            -- Master (A4+), Composition (B5), Medium (A5), etc.
    edition_theme VARCHAR(100),           -- 'Grow Your Mind', 'Year of the Horse', 'Luminous'
    color_name    VARCHAR(50),            -- 'Sunflower', 'Cherry', 'Pumpkin', 'Indigo'
    price         DECIMAL(12,2) NOT NULL,
    stock_status  INT DEFAULT 0,
    personalization_config JSONB,         -- Constraints for "Writing by Hand" or custom text
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

##### CollectibleAttributes Table (Extended Metadata)

```sql
CREATE TABLE collectible_attributes (
    attribute_id    SERIAL PRIMARY KEY,
    product_id      UUID REFERENCES products(product_id),
    collectible_type VARCHAR(50),         -- 'Coins', 'Stamps', 'Trading Cards'
    capsule_system  VARCHAR(50),          -- 'QUADRUM', 'QUICKSLAB', 'Round'
    intercept_tech  BOOLEAN DEFAULT FALSE, -- INTERCEPT® tarnish protection
    supplement_year INT,                  -- 2021 through 2024
    optical_spec    VARCHAR(100),         -- 'UV Lamp HOLMES', 'L81', 'Digital Microscope'
    material_grade  VARCHAR(100)          -- 'VOLTERRA' (Wood), 'NOBILE' (Leatherette)
);
```

##### Region & Locale Table

```sql
CREATE TABLE locales (
    locale_id          SERIAL PRIMARY KEY,
    country_code       VARCHAR(2),        -- US, CA, DE, AT, CH, UK, FR, ES
    language_code      VARCHAR(5),        -- EN, FR, DE, ES
    display_name       VARCHAR(100),      -- e.g., 'Autres Pays - Français'
    currency_code      VARCHAR(3),
    shipping_threshold DECIMAL(10,2)
);
```

#### 3. Product Catalog Logic & Categorization

##### Stationery Taxonomy (Leuchtturm1917)

* **Notebooks:** Master (A4+), Composition (B5), Medium (A5), Paperback (B6+), Pocket (A6).
    * *Special Editions:* 120G, Bauhaus, **Monocle** (Urban Lifestyle), **Luminous** (Bethge Collaboration/Pop Art).
    * *2026 "Grow Your Mind" Theme:* Sunflower, Cherry, Pumpkin, Indigo (1970s inspired).
    * *Year of the Horse:* Limited Edition A5 set with magnetic porcelain tile (Song Dynasty inspiration).
* **Writing Instruments:** Drehgriffel Nr. 1, Drehgriffel Nr. 2 (Pencil), Refills.
* **Journals:** Thesis, Learning, Some Lines A Day, Bullet Journal (Turquoise25).

##### Collectibles Taxonomy (Lighthouse & KABE)

* **Stamps:**
    * *Supplements:* Yearly updates (2021-2024) for Lighthouse and **KABE** brands.
    * *Systems:* OPTIMA, VARIO, GRANDE, MAXIMUM.
* **Coins:**
    * *Storage:* **QUADRUM** Snaplocks, **QUICKSLAB** Slabs, **INTERCEPT®** boxes.
    * *Presentation:* **VOLTERRA** (Wood), **NOBILE** (Leatherette), MB (Acrylic).
* **Optical:** HOLMES UV Lamps, L81 Dual UV Lamp, Digital/Analog Microscopes.

##### Logic Controller Architecture (Strategy Pattern)

```go
// BrandStrategy determines filing paths and brand-specific behavior.
type BrandStrategy interface {
	GetFilingPath(product Product) string
}

func stringOrElse(base, alternate string) string {
	if base == "" {
		return alternate
	}
	return base
}

// CollectiblesStrategy implements BrandStrategy for Lighthouse Publications / KABE.
type CollectiblesStrategy struct{}

func (s *CollectiblesStrategy) GetFilingPath(product Product) string {
	return fmt.Sprintf("Collectibles > %s > %s",
		stringOrElse(product.SubBrand, "Lighthouse"),
		product.CategoryPath,
	)
}

// StationeryStrategy implements BrandStrategy for Leuchtturm1917.
type StationeryStrategy struct{}

func (s *StationeryStrategy) GetFilingPath(product Product) string {
	return fmt.Sprintf("Stationery > %s > %s",
		stringOrElse(product.EditionTheme, "Classic"),
		product.Format,
	)
}
```

#### 4. Functional Requirements: Order Processing & Shipping

The fulfillment engine must differentiate between the two business units for shipping logic and return policies.

##### Shipping & Fulfillment Rules

| Attribute               | Leuchtturm1917                             | Lighthouse Publications                    |
|-------------------------|--------------------------------------------|--------------------------------------------|
| Free Shipping Threshold | $50.00+                                    | $99.00+                                    |
| Warehouse Target        | Flag for processing within 2 business days | Flag for processing within 2 business days |
| Carriers                | FedEx, UPS, USPS                           | FedEx, UPS, USPS                           |
| Methods                 | Priority Mail, Flat Rate                   | Priority Mail, Flat Rate                   |
| Return Policy           | Easy Returns                               | 30-day Return Policy / Easy Returns        |

#### 5. Functional Requirements: Payment & Localization

##### Internationalization (i18n) Mapping

The system must precisely map the following locales as defined in the source footer:

* **North America:** USA (EN), Canada - English (EN), Canada - Français (FR).
* **Europe:** Deutschland (DE), Österreich (DE), Schweiz - Deutsch (DE), Suisse - Français (FR), United Kingdom (EN), France (FR), España (ES).
* **Global Overrides:**
    * "Other Countries - English"
    * "Autres Pays - Français"
    * "Otros países - Castellano"

##### Security & Data Persistence

* **SSL:** Mandatory encryption for all checkout and account-bearing routes.
* **Data Protection:** Implement GDPR and CCPA compliant data retention flags and user "Right to be Forgotten" workflows as linked in the site footers.

#### 6. Back-End Business Logic & Wholesale Modules

##### Specialized Engines

1. **Automatic Supplement Subscription:** A recurring order module for Lighthouse and KABE stamp supplements. Must handle annual release cycles (e.g., triggering 2024 updates).
2. **Personalization Engine:** Manages logic for the "Writing by Hand" brand personalization and custom text embossing on notebooks.
3. **Catalog Request Handler:** Support for 2026 digital (PDF) and physical catalog requests for both stationery and collectibles.
4. **Wholesale Inquiry Workflow:** A dedicated B2B lead generation module routing to the "For Businesses" unit.

#### 7. Implementation Guidance

These descriptions outline the key modules to be implemented in Go.

##### Module 1: Database Schema and Go Types

Define Go structs mirroring the PostgreSQL schema from Section 2. Use a `Product` struct with typed fields for brand, sub-brand, format, edition theme, etc. Create a separate `CollectibleAttributes` struct for specialized fields with a `CapsuleSystem` type (constants: `QUADRUM`, `QUICKSLAB`, `Round`) and an `InterceptTech` bool. Manage schema migrations with raw SQL files.

##### Module 2: Strategy Pattern Shipping Logic

Implement a Go shipping service using the Strategy Pattern. Define a `ShippingStrategy` interface with two implementations: `StationeryShipping` (free over $50) and `CollectiblesShipping` (free over $99). The service evaluates the cart's brand composition and applies the correct threshold. Return a result struct that includes a `WarehouseProcessingDays` field set to 2.

##### Module 3: Faceted Catalog API

Create a chi router handler group for the product catalog. Implement faceted search with pagination. Query parameters must support filtering by brand, format, and edition_theme (e.g., "Year of the Horse" or "Luminous"). Ensure the "Year of the Horse" filter specifically includes the A5 notebook and porcelain tile gift set metadata.

##### Module 4: Localization Middleware

Develop a chi middleware for user localization mapping. It must explicitly support "Other Countries - English", "Autres Pays - Français" (FR), and "Otros países - Castellano" (ES). The middleware detects the region and serves the correct language/currency code based on the locales table defined in Section 2.

##### Module 5: Supplement Subscription Logic

Create a service to manage the "Automatic Supplement Subscription" for stamp collectors. The logic checks for the current year's releases (e.g., 2024 Lighthouse & KABE supplements) and generates recurring orders for users enrolled in the subscription program.

##### Module 6: Personalization Engine

Implement a validation service for product personalization. It handles two types: "Writing by Hand" (brand-standard branding) and custom user text. Use the `personalization_config` JSONB field (mapped to a Go struct) to enforce character limits and font restrictions based on the specific notebook format.

#### 8. Validation & Quality Assurance Constraints

##### Technical Constraints

* **Strict Typing:** All product attributes must be mapped to concrete Go types (structs, typed constants, enums via `iota`). No untyped string constants for domain values.
* **Logic Isolation:** Do not use external libraries for brand-specific shipping thresholds; these must remain configurable system constants.
* **Naming Fidelity:** All trademarks (QUADRUM, QUICKSLAB, VOLTERRA, Drehgriffel, INTERCEPT®) must maintain exact casing and spelling.

##### System Health Check Summary

The system must pass a verification check against these specific source data points:

1. **2026 Palette:** Sunflower, Cherry, Pumpkin, Indigo (1970s / Grow Your Mind).
2. **Collaboration Check:** Verify "Luminous" edition is linked to Bethge and Pop Art metadata.
3. **Collectibles Accuracy:** Ensure KABE and Lighthouse are distinct sub-brands for 2024 supplements.
4. **Specialty Sets:** Confirm "Year of the Horse" gift box logic includes both the A5 notebook and the magnetic porcelain tile.
5. **Localization Check:** Verify the existence of the three "Other Countries" locale variants (EN, FR, ES).
