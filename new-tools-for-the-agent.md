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
