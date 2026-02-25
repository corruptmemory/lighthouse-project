### Research: Commerce/ERP Landscape for the Lighthouse Project

*Compiled 2026-02-25. Context: US subsidiary of German parent company, two
business units (premium stationery + numismatic/philatelic collectibles),
multi-region, subscription model for stamp supplements.*

---

#### The "Improvised Subsidiary" Pattern

When a US subsidiary is stuck between an unresponsive German parent ERP and
daily operational needs, this pattern emerges almost universally:

- **Spreadsheets as system of record** — Excel/Sheets tracking inventory,
  orders, subscriptions. Manual data entry from multiple sources. 1-2 people
  who "know how the spreadsheet works" are single points of failure.
- **Shopify/WooCommerce as storefront** — stood up quickly without IT approval.
  Possibly separate stores per business unit. Subscription plugin bolted on.
  No integration with parent ERP; orders exported as CSVs.
- **QuickBooks/Xero for US financials** — because the parent's SAP doesn't
  produce the reports the US needs. Duplicate data entry. Intercompany
  transactions handled via email.
- **Email + spreadsheets for subscriptions** — renewal reminders sent manually
  or via basic email automation.
- **Point solutions** — ShipStation for fulfillment, Stamps.com for postage,
  maybe a separate inventory tool.

**Where it breaks down:**

1. Data silos and reconciliation hell (month-end close becomes multi-day).
2. Subscription management at scale (~200-300 subscribers is the spreadsheet
   ceiling — missed renewals, no dunning for failed payments).
3. Multi-currency chaos (USD operations, EUR reporting to parent).
4. Inventory visibility gaps (overselling, stockouts).
5. Compliance risk (US sales tax nexus post-Wayfair, potential EU VAT).
6. Key person dependency.

---

#### Solution Categories

| Solution | First-Year Cost | Timeline | Best For |
|---|---|---|---|
| SAP S/4HANA Cloud | $250K-$700K | 9-18 months | If parent is driving & paying |
| SAP Business One | $50K-$100K | 2-4 months | Simpler, but weak e-commerce & subscriptions |
| Oracle NetSuite | $150K-$250K | 4-6 months | Multi-currency/multi-subsidiary sweet spot |
| Odoo | $30K-$90K | 2-4 months | 80% capability at 20% price |
| Shopify Plus + Chargebee | $50K-$150K | 1-3 months | Fastest, but still need financial back-end |

##### Full Cloud ERP

**Oracle NetSuite OneWorld** — Native multi-currency and multi-subsidiary.
Strong financial consolidation (critical for reporting to German parent).
Built-in subscription billing (SuiteBilling). Cloud-native. Pricing:
~$150K-$250K first year including implementation, ~$80K-$150K/year ongoing.
Pre-built SAP connectors via Celigo.

**Microsoft Dynamics 365 Business Central** — Strong if already a Microsoft
shop. Good inventory/warehouse management. No native e-commerce — need
third-party storefront. No native subscription billing. Pricing: ~$50K-$80K/year
licensing + $40K-$100K implementation.

**Odoo** — Full suite (CRM, e-commerce, inventory, accounting, subscriptions)
at dramatically lower cost. Open source core. Weaker multi-subsidiary
consolidation than NetSuite. Thinner US partner ecosystem. Pricing: ~$10K-$15K/year
licensing + $20K-$75K implementation.

##### Commerce Platforms (not full ERP)

**Shopify Plus** — Speed to market (weeks, not months). Excellent storefront.
Subscription billing via third-party apps only (ReCharge ~$300-500/month).
Multi-currency subscriptions are painful. Two business units may need two
stores. Pricing: ~$2,300/month + implementation.

**BigCommerce Enterprise** — More API-friendly than Shopify. Better for complex
catalogs. Better native multi-currency. Smaller app ecosystem. Pricing:
~$1K-$3K/month enterprise tier.

##### Headless Commerce / API-First

**commercetools** — Maximum flexibility, MACH architecture. Enterprise-grade.
Strong multi-currency/multi-locale. No storefront included. No native
subscription billing. Pricing: $50K-$200K+/year before development.

**Medusa** (open source) — Free, Node.js-based. Full control. Significant
development overhead. No native subscriptions.

**Swell** — Headless with native subscription billing (key differentiator).
Zero transaction fees. Smaller ecosystem.

##### Subscription Billing (alongside other systems)

**Chargebee** — Purpose-built for subscriptions. Handles trials, dunning,
proration. Integrates with Shopify, NetSuite. Pricing: from $599/month.

**Zuora** — Enterprise-grade. Deep SAP integration. Expensive and slow to
implement ($50K+/year, 3-12 months).

##### iPaaS / Middleware

**Celigo** (~$1K-$3K/month) — Pre-built flows for NetSuite, Shopify, SAP.
Low-code. Purpose-built for e-commerce/ERP integration.

**Boomi** (~$2K/month) — Broad connector library including SAP. Low-code.

**MuleSoft** ($50K+/year) — Enterprise-grade. Strongest SAP connectors.
Overkill for mid-market.

---

#### SAP Specifics

**Which product:** S/4HANA Cloud Public Edition is the only strategic choice.
Business One is too limited. **Business ByDesign is being delisted from the
pricelist for new customers starting April 2026** — do not pursue.

**Realistic timeline:** 9-18 months from decision to stable operation.

**Cost:** $250K-$700K first year, $80K-$200K/year ongoing.

**Common failure modes:**

1. "Automating chaos" — replicating manual processes instead of adopting
   SAP's standard workflows.
2. Scope creep from parent company inserting requirements that serve their
   reporting needs but hurt US operations.
3. Underestimating change management — the 5-10 people running the US
   operation resist imposed workflows and build shadow systems.
4. Integration underestimation — connecting US S/4HANA to parent's SAP
   (possibly older ECC) consumes 30-50% of budget.
5. "We'll do e-commerce later" — SAP is not a commerce platform, you need a
   storefront on day one.
6. Subsidiary deprioritized — parent treats it as a template rollout,
   ignoring US-specific tax rules, fulfillment, and business models.

---

#### Strategic Question

**Is SAP a real near-term requirement, or an aspiration?**

- If SAP is landing in 12 months → build Go system as front-end/middleware
  that will eventually defer to SAP for financials and inventory.
- If SAP is 2+ years out → build Go system as the authoritative back-end,
  with clean interfaces that could connect to SAP later.
