---
status: proposed
---

# A venue's own products, mapped from each supplier's articles; price moves compare within one supplier

The demo treats a product as an exact name shared by invoices, price history and recipes, and compares each price against the last price from any supplier. For real invoices we split this in two: a **Product** is the venue's own name and base unit (what recipes use), and a **Supplier Article** is one supplier's printed name, unit and pack size, mapped to a product once at review and reused afterwards. A **Price Move** compares against the last price of the same supplier article, because "this supplier charged me more than last time" is the overcharge question; the cheapest other supplier is shown only as context. The **Current Cost** used for margins is the latest price by invoice date from any supplier.

## Considered Options

- **Exact names (today):** breaks on the first renamed or differently printed article, silently dropping lines from margins.
- **Automatic fuzzy matching without confirmation:** fast, but a wrong match corrupts costs invisibly.
- **Cross-supplier price moves (today):** flags a supplier switch as a price rise, as the demo does for `BS-2207.pdf`.

## Consequences

Every price point, alert and cost hangs off this mapping, so changing it later means migrating all history. Recipes move to base units (cl, g, pieces) with pack sizes on supplier articles, replacing the demo's "fraction of a bottle" quantities.
