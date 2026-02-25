### Domain Expert Questions: Lighthouse Back-End System

These questions are designed to validate and deepen the technical specification,
which was inferred from the customer-facing websites (lighthouse.us and
leuchtturm1917.us). The spec captures what a customer *sees* — these questions
aim to uncover what the business *does*.

Answers should be recorded inline. Anything marked with **[ASSUMED]** is a guess
from the spec that needs confirmation or correction.

---

#### 1. Current State — What Exists Today

Before building anything, we need to understand what we'd be replacing,
supplementing, or integrating with.

- What systems does the US office use today for order management? (e.g., Shopify,
  custom platform, SAP, spreadsheets, a mix?)
- Are lighthouse.us and leuchtturm1917.us backed by the same system or are they
  separate platforms?
- What's the relationship between the US office and the German parent company in
  terms of systems? Does the US office run its own infrastructure or does it plug
  into a global system?
- What's the most painful manual process in daily operations?
- What do customers contact support about most often?

---

#### 2. Product Catalog

The spec infers a product taxonomy from website navigation. Several details need
validation.

- **[ASSUMED]** The two top-level brands are "Leuchtturm1917" (stationery) and
  "Lighthouse Publications" (collectibles). Is that the actual business
  distinction, or is it organized differently internally?
- **[ASSUMED]** "Monocle" and "Bethge" are listed as sub-brands. Are these
  actually sub-brands, or are they collaboration/partner lines with different
  commercial terms?
- **[ASSUMED]** "Trading Cards" appears in the collectible types. Does Lighthouse
  Publications actually sell trading card supplies, or was that inferred
  incorrectly?
- How are products managed today — a PIM (product information management) system,
  spreadsheets, direct database edits, the e-commerce platform's admin?
- How often does the catalog change? Are there seasonal refreshes, or is it
  continuous?
- How many SKUs are there roughly? Hundreds? Thousands? Tens of thousands?
- Are there products that appear on one regional site but not another?

---

#### 3. Orders & Fulfillment

The spec describes shipping rules but not the operational flow behind them.

- Walk through what actually happens when a customer places an order, step by
  step — from the moment they click "buy" to the package leaving the warehouse.
- **[ASSUMED]** Free shipping thresholds are $50 (stationery) and $99
  (collectibles). Are those accurate? Do they change by region or during
  promotions?
- Can a single order contain both stationery and collectibles? If so:
    - Which shipping threshold applies?
    - Does it ship as one package or two?
    - Does it come from one warehouse or multiple?
- Where is inventory physically located? One US warehouse? Multiple? Does
  anything ship directly from Germany?
- **[ASSUMED]** Carriers are FedEx, UPS, USPS. How is the carrier selected — by
  customer choice, by the system automatically, or by warehouse staff?
- What happens when an item is out of stock at the time of order? Backorder?
  Cancel the line item? Hold the whole order?

---

#### 4. Subscriptions (Stamp Supplements)

The spec mentions an "Automatic Supplement Subscription" in one sentence. This is
potentially the most complex module to build, so it needs the most detail.

- How do supplement subscriptions work today? Is there an automated system, or is
  it managed manually?
- When a new supplement year is released (e.g., 2024 KABE supplements), what
  triggers the process? A person? A date? A supplier notification?
- Are subscribers auto-charged, or do they receive an invoice/notification first?
- What payment method is used for recurring charges?
- Can a subscriber be on multiple subscriptions (e.g., both Lighthouse and KABE
  supplements)?
- What happens when a subscriber wants to cancel, pause, or skip a year?
- Roughly how many active subscribers are there? (Order of magnitude is fine —
  tens, hundreds, thousands?)
- Are supplements always annual, or are there other cadences?

---

#### 5. Personalization & Embossing

The spec mentions "Writing by Hand" and custom text but doesn't describe the
physical process or constraints.

- What exactly is "Writing by Hand"? Is it a specific font/branding style
  applied to the notebook, or something else?
- What products can be personalized? Only notebooks, or also journals and other
  items?
- What are the actual constraints? (Character limits, allowed characters, font
  choices, placement options?)
- Does personalization happen in-house at the US warehouse, or is it outsourced /
  done at the factory?
- Does a personalized order take longer to fulfill? If so, how much longer?
- Can a personalized item be returned?

---

#### 6. Localization & Regional Sites

The spec lists locales from the website footer, but it's unclear how they map to
actual backend behavior.

- The footer lists ~12 locale options (US, Canada EN/FR, several European
  countries, three "Other Countries" variants). Does the US office serve all of
  these, or only the North American ones?
- Are prices set independently per region, or converted from a base price in
  EUR/USD?
- Is the product catalog identical across all regions, or do some regions have
  products the others don't?
- How is currency/tax handling done today?
- The "Other Countries" options — do those actually produce different behavior
  (different shipping rates, different product availability), or are they just
  language switches?

---

#### 7. Wholesale / B2B

The spec mentions a "Wholesale Inquiry Workflow" but gives no operational detail.

- How does a wholesale inquiry come in today? Web form? Email? Phone?
- What happens after the inquiry — does someone manually follow up, or is there
  a CRM / pipeline?
- Do wholesale customers place orders through the same website, a separate
  portal, or by phone/email?
- Are wholesale prices a fixed discount off retail, or negotiated per account?
- Roughly what fraction of revenue is wholesale vs. direct-to-consumer?

---

#### 8. Inventory & Supply Chain

The spec has no inventory model at all — this section is entirely unknown.

- Is inventory tracked in real time, or updated in batches?
- Who/what is the source of truth for inventory counts?
- How does the US office receive new inventory — directly from factories, via
  the German parent, or through distributors?
- Are stationery and collectibles stocked in the same warehouse?
- Are there items that are made-to-order vs. stocked?
- How are discontinued products handled? Do they stay visible on the site until
  stock runs out?

---

#### 9. Scope & Priority

These are less about domain knowledge and more about deciding what to build first.

- If you could wave a magic wand and fix one thing about the current system,
  what would it be?
- Which of these areas causes the most day-to-day friction: product catalog
  management, order fulfillment, subscription handling, localization, or
  something else entirely?
- Are there internal tools or dashboards that staff wish they had?
- Is there a reporting or analytics need that's currently unmet?
