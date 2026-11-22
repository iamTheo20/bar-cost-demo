# MVP plan: from demo to a sellable product for small bars and restaurants

Derived from `docs/grilling-log.md` (rounds R1–R7). Every decision there was
self-answered overnight and is marked **ASSUMED — review**. Terms are as in
`GLOSSARY.md`; the hard-to-reverse choices are ADRs 0002–0006 in `docs/adr/`.

## Assumptions to confirm

The most consequential self-answered decisions. If any is wrong, the phases
below change.

1. **Customer and market** (R1-Q1, R1-Q2): independent single-site bars and
   restaurants in Sweden, SEK only; small groups later.
2. **The promise** (R1-Q3, ADR 0006): one loop, invoice → checked → reviewed
   → price alerts and below-target menu items. No inventory, ordering, POS or
   accounting export in the MVP.
3. **Human accept** (R4-Q5, ADR 0004): no invoice counts until a person
   accepts it; no auto-accept.
4. **Products vs supplier articles** (R3-Q1, R3-Q2, ADR 0003): the venue's own
   products, mapped from each supplier's printed articles; price moves compare
   within one supplier. This changes today's cross-supplier comparison.
5. **Base units and pack sizes** (R3-Q3): recipes move from "0.0571 bottle" to
   "4 cl"; existing recipe CSVs must be converted.
6. **Extraction via the Anthropic API** (R4-Q4, R5-Q5, ADR 0002): server-side,
   model chosen by an eval on ≥ 50 real invoices; budget ≤ USD 0.05 per invoice
   on average.
7. **SQLite in one Go binary on one EU server** (R5-Q1, R5-Q4, ADR 0005), not
   managed Postgres.
8. **Pricing** (R6-Q1): flat per venue per month, starting point 495 SEK ex
   VAT, two free pilot months.
9. **Success bar** (R1-Q5): 3–5 Stockholm pilots for 4 weeks, ≥ 80% of
   invoices accepted with no edits beyond product matching, ≥ 2 pilots willing
   to pay.

## Where the demo stands (facts, see grilling log F1–F13)

Working and tested in Go: rule-based reading of the demo text layout, Claude
reading of PDFs/photos via the local CLI, exact arithmetic check, price moves
against history, margin before/after against per-item targets. Missing for a
product: storage, accounts, web UI, product matching, units, VAT, line kinds,
duplicates, server-side LLM, deployment, billing.

## Phases

Effort is in focused developer-days for one developer who knows the code,
rounded up. Build ≈ 40 days, plus ≈ 4 developer-days spread over a 4-week
pilot: ≈ 44 days in all, roughly 10–11 calendar weeks for one developer.

### Phase 0: Finish the Go switch (1 day)

Tasks
- Replace the Python harness (list in the Go-rewrite pull request): Makefile
  targets `check` = `gofmt -l` empty + `go vet ./...` + `go test ./...`, and
  `verify` = `go test ./cmd/barcost -run TestFeaturesMD`; CI with
  `actions/setup-go`; drop `pyproject.toml`, `ruff.toml`, `uv.lock` and the
  Python rules engine or port its rules.
- Delete `src/`, `tests/`, `scripts/` (the overnight run was not allowed to).
- Add `staticcheck` or `golangci-lint` to `check`.

Acceptance
- `make check verify` and CI are green on Go only; no Python files remain
  except `testdata/golden/capture.py` (provenance).

### Phase 1: Domain model and storage (8 days)

Tasks
- SQLite schema with migrations: Organisation, Venue, Member (Owner/Staff),
  Supplier, Product (base unit), Supplier Article (pack size, mapping),
  Invoice (state, file key, hash, VAT totals), Invoice Line (kind), Credit
  Note, Price Point, Menu Item (price ex VAT, VAT rate, target margin), Recipe
  Ingredient (base-unit qty), Dispute status. Every venue-scoped table carries
  the venue.
- Move core logic onto the stored model: price move vs last price of the same
  supplier article; current cost by latest invoice date; serving cost in base
  units; line kinds (only product lines make price points); rounding line
  ≤ ±1.00 SEK; duplicate refusal by (supplier, number) and file hash.
- CSV import and export (menu, recipes, products, price history) with the
  base-unit columns; a converter for the demo's current CSVs.
- Configurable price-alert threshold per venue and per product.
- Keep the golden and feature tests passing on the demo data through the new
  code paths, updated only where ADR 0003 changes behaviour on purpose
  (cross-supplier comparison) and documented in `FEATURES.md`.

Acceptance
- Demo data imports; `invoice`, `prices`, `margin` give the same numbers from
  the database as today, except the documented ADR 0003 change.
- Uploading the same invoice twice is refused; a backdated invoice does not
  replace a newer current cost; a deposit line never becomes a price point.
- Table-driven tests for every rule above.

### Phase 2: Invoice ingestion and review (12 days)

Tasks
- Web upload (multi-file, phone camera) and per-venue forwarding address via
  an inbound-email provider's webhook; files to EU object storage.
- Background job queue (in-process, persisted in SQLite) for extraction.
- Anthropic API extractor behind the existing `Extractor` interface: schema
  extended with line kind and VAT; same prompt rules; Go-side validation;
  timeout 60 s, max 2 attempts, fallback to Needs Manual Entry; cost, tokens
  and latency logged per call and per venue.
- Extraction eval harness: ≥ 50 labelled real invoices from pilots, accuracy
  per field and per model; pick the cheapest model meeting ≥ 98% exact product
  lines.
- Review screen: original beside the draft, editable lines, product matching
  with proposals (exact, then normalised name), arithmetic issues
  highlighted, Accept / Reject, reopen by Owner.
- Manual entry form for Needs Manual Entry.

Acceptance
- A forwarded PDF and a phone photo each reach Draft without a developer.
- Prompt-injection test invoice ("ignore previous instructions…") is
  extracted as data.
- The eval report exists and names the chosen model and its measured cost per
  invoice.
- A clean invoice can be reviewed and accepted in under 30 seconds by a
  first-time user (timed in a hallway test).

### Phase 3: Alerts, margins and the digest (6 days)

Tasks
- Invoice page panels: arithmetic issues (with copyable dispute text and
  Disputed/Resolved status), price moves and alerts, affected menu items with
  margin before/after and SEK per serving.
- Menu overview: every menu item's margin vs target, below-target first.
- Recipe and menu editing; VAT-inclusive price entry converted by item rate.
- Daily digest email, only when something is new.

Acceptance
- With the demo data loaded as a venue, the digest after accepting `SF-1093`
  names Cheddar's arithmetic issue, the Avocado alert and Guacamole & chips
  below target, matching the CLI's numbers.
- No digest is sent on a day with nothing new.

### Phase 4: Accounts, tenancy and billing (8 days)

Tasks
- Magic-link sign-in, session cookies, CSRF protection, invites, Owner/Staff
  permissions; every query scoped by venue, with a test that a member of one
  organisation cannot read another's data.
- Hosted subscription billing in SEK with a customer portal; trial for
  pilots; fair-use invoice volume check.
- Terms, privacy policy, DPA with sub-processor list (lawyer-reviewed);
  account deletion and full data export (CSV + original files).

Acceptance
- Cross-tenant access tests pass for every route.
- A new organisation can sign up, subscribe, invite Staff and cancel without
  developer help.

### Phase 5: Deploy, operate, onboard (5 days)

Tasks
- Container build of the single binary; EU-hosted production and staging;
  TLS; continuous SQLite replication to EU object storage with a daily
  automated restore check.
- `log/slog` JSON logs, uptime check, error alerts, weekly extraction-accuracy
  and LLM-cost report per venue (alert above 10% of subscription).
- Concierge onboarding kit: spreadsheet-to-CSV templates, a 60-minute session
  script.

Acceptance
- Restore from backup to a fresh server in under 30 minutes, rehearsed.
- Deploy is one command and takes under a minute of downtime.

### Phase 6: Pilot (4 weeks calendar, ~4 days of developer time)

Tasks
- Onboard 3–5 Stockholm venues; process all their invoices; weekly check-ins;
  fix top issues.
- Measure: share of invoices accepted with no edits beyond matching; review
  time; LLM cost per invoice; arithmetic issues and alerts found; willingness
  to pay at 495 SEK.

Acceptance (the MVP is sellable when)
- ≥ 80% of invoices accepted with no edits beyond product matching.
- Every pilot saw at least one real issue or below-target item.
- ≥ 2 pilots agree to pay; LLM cost per invoice within budget.

## Risks

| risk | effect | mitigation |
| --- | --- | --- |
| Extraction accuracy on real Swedish invoices and phone photos is below 98% | review becomes data entry; pilots churn | eval set before choosing a model; Needs Manual Entry path; measure edits per line weekly |
| Product matching is tedious at the start | first weeks feel like work | concierge onboarding processes the first month; reuse mappings per supplier article |
| Venues do not maintain recipes | margins are wrong or empty | import their existing spreadsheet; start with the top 20 menu items |
| LLM cost per invoice above budget, or provider price changes | margin of the subscription shrinks | cheapest-passing model; per-venue cost alerts; files deduplicated before extraction |
| Prompt injection or malicious files | wrong data, data leak | no tools beyond reading the file; schema-bound output; Go validation; human accept (ADR 0004) |
| Single server, single process (ADR 0005) | downtime on failure or deploy | continuous backup, rehearsed restore; Postgres migration path kept portable |
| Willingness to pay at the assumed price is low | no business | test price in pilot; annual plan; consider per-group pricing |
| Scope creep into inventory, POS or accounting | MVP slips | ADR 0006; accounting export is the first post-MVP item, not an MVP item |
| Harness still Python after the rewrite | CI red, gates confused | Phase 0 first |
| Legal (GDPR, DPA, bookkeeping expectations) | cannot sign customers | lawyer-reviewed documents in Phase 4; state clearly we are not the legal archive |

## Out of scope for the MVP (ADR 0006)

POS integration and sales mix; inventory and stock counts; purchasing and
ordering; accounting/bookkeeping export (first after the MVP); acting as the
legal bookkeeping archive; Peppol/EDI e-invoices; auto-accept;
multi-currency and non-Swedish VAT; native mobile apps; recipe yields, waste
and sub-recipes; labour cost; supplier price lists and portals; an accountant
role; sending emails to suppliers.
