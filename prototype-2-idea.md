# Prototype 2 Ideas

## Trust and Confidence
We have been discussing the "trust and confidence" issue a fair bit;
and a tough nut it is in general. Calling out SQL as we go is
certainly a start, but I think we need to go further: a "visualizer"
for the queries in question will be needed.

The basic idea is wherever we call out SQL there should be a link,
something like "explore" or "visualize" that brings you to a full
screen experience where the SQL in question is on display. Similar to
Excel's ability to highlight the cells related to a formula, the SQL
is highlighted, or sections highlighted with a mouse over, that when
clicked can show samples of the "data set" the component of SQL is
considering.

For example:

``` sql
SELECT p.first_name, p.last_name, pn.phone_number, pn.phone_number_kind 
FROM contacts p
JOIN phone_numbers pn 
  ON pn.contact_id=p.id
WHERE 
  p.last_name="Smith";
```

- If you were to mouse over `contacts` you could click on it and see, say 5 rows from the `contacts` table.
- Mouse over the JOIN and the entire JOIN clause is highlighted and if you click on it you'll see say 5 rows of `contacts` joined with `phone_numbers`.
- Mouse over ONLY `phone_numbers` and you can click into see say 5 rows of `phone_numbers`
- Mouse over the SELECT or WHERE and the entire SELECT statement is lit up and you can see 5 rows of the results.

The idea is that there's an easy way to explore what the AI wrote as the select statement and how the pieces fit together to form the whole.

Of course, on this screen would be a prompt window where questions
could be asked. Even the SQL could be altered (for now by the AI, but
eventually by either the human or AI).

## Widget "Development" vs Dashboard "Development"

At the moment we have a single page where the user can interact with
agent to produce "widgets."  I'm thinking that things work like this:

On the one page we have, that's a "Dashboard" context.  The user can say stuff like:

> I want a dashboard that summarizes:
>  * Sales for the last 30 days
>  * Summarizes order status for the last 30 days
>  * Summarizes orders shipped by carriers in the last 30 days

What happens is:
* If a widget for the "view" being asked for (like "Sales for the last
  30 days") exists, drop it in place on the dashboard
* If we don't have a widget defined for the request drop in a
  "placeholder" that says "Create widget for: '...'".  If the user
  clicks on it they are taken to a full screen where they can interact
  with the AI to develop the widget.  When they are happy that widget
  gets dropped into place
