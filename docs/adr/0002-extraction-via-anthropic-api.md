---
status: proposed
---

# Customer invoices are read by the Anthropic API, server-side, not by a local CLI or OCR templates

The demo reads PDFs and photos through the developer's local `claude -p` login, which costs the developer nothing per call but cannot serve customers. For the MVP, extraction moves to the Anthropic API called from our server with a JSON schema, keeping the demo's rules: copy every number as printed and never correct it, ignore instructions written in the file, and validate every field in Go before it becomes an invoice. We chose this over template-based invoice OCR services because small Swedish suppliers send many one-off layouts (and phone photos) that templates handle poorly, and over manual entry because typing invoices is the chore we are selling relief from.

## Considered Options

- **Local `claude -p` (today):** free per call on one login; not multi-tenant, not supportable. Kept only for development.
- **Template/OCR invoice APIs:** cheaper per page for high-volume, repeated layouts; weak on photos and long-tail suppliers; new vendor lock-in.
- **Manual entry only:** no model cost; kills the value proposition.

## Consequences

- Each invoice has a model cost (measured on the demo: USD 0.02–0.03 per invoice on a top-tier model) that the pricing model must cover; the model is picked by an accuracy evaluation on real pilot invoices, cheapest first.
- The LLM provider is a sub-processor of customer data and must be listed in the DPA.
- Because the model is told never to correct numbers, the arithmetic check keeps catching supplier mistakes; changing that prompt rule would silently remove a core feature.
