# New "tools" for the AI agent

## Query tools

A tool called `order_search` returns the following:

- `order_id`
- `order_number`
- `customer_name`
- `customer_email`
- `brand_mix`
- `total REAL`
- `item_count`
- `shipping_carrier`
- `shipping_method`
- `status TEXT`
- `order_date`
- `created_at`
- `order_items` <- this is a JSON array of order items
  - Example: ```json
  [
    {
      "item_id": 223,
      "order_id": "LH-1234",
      "product_id": 1234,
	  // JOIN on products and IN-LINE key values from the `products` table
      "quantity": 5,
      "unit_price": 25.00,
      "line_total": 125.00,
    },
	...
  ]
  ```

It takes a set of key/value pairs (JSON is fine):

- `order_number` -- this corresponds to the human-readable Order ID (which is unique and effectively shadows the `order_id` column).  Either a single order id can be supplied or an array of order ids. examples
  - `order_number=LH-4534` - single match
  - `order_number=[LH-4534, LH-3345]` - "IN query"
- `customer_name` -- a string, with possible "wildcards" (globs or regular expressions)
- `customer_email` -- a string, with possible "wildcards" (globs or regular expressions)
- `brand_mix` -- a string, with possible "wildcards" (globs or regular expressions)
- `shipping_carrier` -- a string, with possible "wildcards" (globs or regular expressions)
- `shipping_method` -- a string, with possible "wildcards" (globs or regular expressions)
- `status` -- an array of strings, to be interpreted as an IN query
- `order_date` -- a string.  The string encodes an expression.  Examples include:
   - Equals: `= 2026-02-01`
   - Not Equals: `!= 2026-02-01`
   - Relative comparison: `> 2026-02-01`, `< 2026-02-01`, `>= 2026-02-01`, `<= 2026-02-01`
   - Between (inclusive): `between(2026-02-01,2026-02-10)`
   - In: `[2026-02-01,2026-02-10]`
- `created_at` -- a string.  Uses the same expressions as `order_date`
- `contains_product` -- a "nested" key-value (JSON is fine) structure.  Queries can include:
   - `brand` - a string, can be a single brand or an array in `[]` to be used as an IN query
   - `sku` - a string, can be a single sku, or an array of skus in `[]`
   - `name` - a string.  Can contain "wildcards" (globs or regular expressions)

### Sorting

- `sort` -- a string specifying column and direction, defaulting to
  `order_date desc`. Multiple sort keys separated by commas.
  - `sort=order_date desc` -- single sort
  - `sort=status asc, total desc` -- multi-column sort
  - Valid columns: any top-level return field (`order_number`, `customer_name`,
    `brand_mix`, `total`, `item_count`, `status`, `order_date`, `created_at`).
    The Go layer whitelists these to prevent injection.

### Pagination

- `limit` -- integer, maximum number of results to return. The Go layer
  enforces a ceiling (e.g. 200) regardless of what the agent requests.
- `offset` -- integer, number of results to skip before returning.

**Caveat:** limit/offset pagination is unreliable in high-throughput live
systems. When new orders arrive between page fetches, rows shift — the user
sees duplicates or misses rows entirely, and it looks like "pagination is
broken." This is fine for prototyping and low-volume use, but production
systems with significant order flow will need cursor-based pagination
(keyset pagination using `WHERE order_id > last_seen_id ORDER BY order_id`)
or snapshot isolation. Noting this now so we don't get surprised later.

## Order manipulation tools

TODO — next design iteration. The goal is structured mutation tools
(`update_order`, etc.) that replace raw DML in the bulk operations flow.
The agent proposes structured changes, the human reviews readable
descriptions instead of SQL strings, same propose-review-execute trust
boundary.
