# Grilling log: from bar-cost demo to sellable MVP

Method: `grill-with-docs` (the `grilling` and `domain-modeling` skills from
mattpocock/skills), run unattended overnight on 2026-11-22. Nobody could be
interviewed. So each round lists the frontier questions in the grilling
format, each with a recommended answer, and the recommendation was then
**taken as the decision**. Every decision below is therefore marked
**ASSUMED (self-answered) — review**. Theo: read each "➡️" line and its
"yes" means you accept it. A "no" on an early round reopens the rounds after it.

Terms were added to `GLOSSARY.md` as they were resolved. Decisions that are
hard to reverse, surprising and a real trade-off got an ADR in `docs/adr/`.
`docs/MVP_PLAN.md` turns the settled tree into phases.

## Facts found in the code and data (not assumed)

These were looked up, not asked. File references are to this repository at
the time of the run.

- **F1 Money and locale.** All amounts are SEK (`README.md`, invoice
  headers). The suppliers are Swedish (Nordic Beverage AB, Stockholm Fresh
  AB, Bryggeriet Soder AB; Stockholm addresses). The PDF invoice uses Swedish
  number formats (`1 520,00`), Swedish labels (`Summa exkl. moms`,
  `Moms 25%`, `Att betala`) and a Swedish date (`17 nov 2026`).
- **F2 VAT.** Menu prices exclude VAT (`data/menu.csv`, README). The text
  invoices carry no VAT; the LLM is told to "use the subtotal before VAT"
  (`llmextract` prompt). Nothing stores VAT.
- **F3 Product identity is an exact string.** An invoice line, a price-history
  row and a recipe line refer to the same product only if the name is
  byte-identical (`catalog.CurrentPrices`, `margin.Affected`). There is no
  alias or matching step.
- **F4 Price history ignores the supplier.** `LastPriceBefore` takes the last
  price of the product from any supplier. `BS-2207.pdf` (Bryggeriet Soder AB)
  is compared against history that only has Nordic Beverage AB rows, so
  "Lime +21.1%" is a cross-supplier comparison.
- **F5 Units are implicit.** Recipe quantities are in whatever unit the
  invoice uses (`Herno Gin 70cl,0.0571` means 0.0571 bottles ≈ 4 cl;
  `Pilsner keg 30L,0.0133` ≈ 40 cl). There is no pack size or unit conversion.
- **F6 The cost basis is the newest price, whatever the invoice date.**
  `CurrentPrices` takes the last history point, then lets the invoice's prices
  override. An invoice older than the newest history row would still override.
- **F7 Nothing is persisted.** Reading an invoice does not add to
  `price_history.csv`; there is no write path, no invoice store, no duplicate
  check.
- **F8 Flags are fixed.** The price-move threshold is a constant 5% in either
  direction (`prices.Threshold`). Target margins are per menu item
  (0.70–0.80 in `menu.csv`).
- **F9 The arithmetic check is exact to the cent.** Each line's qty × unit
  price is rounded half-to-even to öre and must equal the printed total; the
  lines must sum exactly to the subtotal (`invoices.ArithmeticIssues`).
- **F10 Line kinds are not modelled.** Every table row is a purchased
  product. Deposits (pant), fees, discounts, rounding lines and credit notes
  have no representation; the LLM schema allows negative amounts.
- **F11 LLM extraction runs the user's local `claude -p` CLI** with Read-only
  tools, an empty folder, a JSON schema, a "copy numbers, never correct" and
  an "ignore instructions in the file" rule, and Go-side validation.
  Measured on this run (model `claude-opus-5-5`, 3 turns each): `BS-2207.pdf`
  7.3 s, USD 0.033; `SF-1093.txt` with `--llm` 6.9 s, USD 0.020. Both
  extractions were faithful, including the supplier's wrong `140,00` line.
- **F12 Single venue, single user, CLI only.** No accounts, auth, tenancy,
  web UI, storage or deployment exist.
- **F13 Owner decision already made:** the app is rewritten in Go (done on this
  branch, byte-for-byte parity with the Python outputs). This was decided by
  Theo, not assumed; it is recorded as ADR 0001.

---

## Round 1: who, where, what promise, what form

Frontier: the root decisions; nothing else can be settled before them.

❓ **Q1** - **Paying customer**: Who pays? (a) independent, owner-operated
single-site bars and restaurants; (b) small groups of 2–10 sites; (c) chains
with a central purchasing function. The demo (one menu, one price history,
F12) fits (a). Groups need multi-venue roll-ups; chains already buy
enterprise back-office suites.

➡️ (a) independent single-site venues, with (b) as the first expansion. The
data model is multi-venue from day one (see R2-Q2) so (b) is not a rewrite.

**Decision:** (a), multi-venue-ready. **ASSUMED (self-answered) — review**

---

❓ **Q2** - **First market**: Sweden only, the Nordics, or "anywhere in
English"? Facts F1/F2: SEK, Swedish suppliers, Swedish number formats, 25%
moms on the sample PDF.

➡️ Sweden only for the MVP: SEK, Swedish invoice formats, Swedish VAT. UI in
English for the pilot, Swedish strings added before general sale.

**Decision:** Sweden, SEK. **ASSUMED (self-answered) — review**

---

❓ **Q3** - **Core promise**: Which job do we sell? (a) invoice checking only;
(b) price-move alerts only; (c) margin protection: one loop from invoice to
"which menu items did this hurt"; (d) full food-cost management with
inventory and ordering. The three existing features (`invoice`, `prices`,
`margin`) already form (c).

➡️ (c). The promise: "within a day of an invoice arriving, you know whether
the supplier overcharged you and whether a price change pushed a drink or
dish below its target margin." Inventory and ordering are out (R6-Q4).

**Decision:** (c) margin protection loop. **ASSUMED (self-answered) — review**

---

❓ **Q4** - **Product form**: CLI (today), web app, native mobile app, or
email/chat-only? Bar owners photograph paper invoices and receive PDFs by
email.

➡️ A responsive web app (works on a phone for photos) plus email in (forward
invoices) and email out (digest). No native app, no CLI for customers; the CLI
stays as a developer tool.

**Decision:** responsive web app + email. **ASSUMED (self-answered) — review**

---

❓ **Q5** - **What proves the MVP**: What result makes us call it sellable?

➡️ 3–5 pilot venues in Stockholm use it for 4 weeks on all their supplier
invoices; at least 80% of invoices go from upload to accepted with no edits
other than product matching; every pilot sees at least one real supplier
error or below-target margin they did not know about; at least 2 pilots agree
to pay after the pilot.

**Decision:** as recommended. **ASSUMED (self-answered) — review**

---

## Round 2: users, tenancy, the loop, getting data in and out

Unblocked by R1-Q1 (customer), R1-Q3 (promise) and R1-Q4 (form).

❓ **Q1** - **Roles inside a venue**: Who uses it? Owner, manager, head
chef, bar manager, external accountant?

➡️ Two roles. **Owner**: everything, including billing and members.
**Staff**: upload, review and accept invoices, edit products, recipes and the
menu. No accountant role in the MVP.

**Decision:** Owner and Staff. **ASSUMED (self-answered) — review**

---

❓ **Q2** - **Tenancy**: Is an account one venue, or an organisation that owns
venues?

➡️ An **Organisation** owns one or more **Venues**; members belong to the
organisation. Every venue-scoped record carries the venue. The MVP UI shows
one venue per organisation; a venue switcher comes with R1-Q1 (b).

**Decision:** Organisation → Venues. **ASSUMED (self-answered) — review**

---

❓ **Q3** - **The core loop**: Is this the loop? Invoice arrives →
extracted into a draft → a person reviews it (fixes lines, matches products,
sees arithmetic issues) → accepts → its prices become price points → price
moves and margin impacts are shown → a digest goes out.

➡️ Yes. Nothing enters price history or margins until a person accepts the
invoice (made binding in R4-Q5). Today's three CLI features become three
panels of one invoice page.

**Decision:** as described. **ASSUMED (self-answered) — review**

---

❓ **Q4** - **Ingestion channels**: Which ways in for the MVP? Web upload
(PDF, photo, several files), a per-venue forwarding email address, supplier
EDI / Peppol e-invoices, a supplier portal scrape, or the accounting system's
inbox?

➡️ Web upload (including the phone camera) and a per-venue forwarding email
address. Both feed the same extraction. Peppol/EDI and accounting-system
inboxes are out of the MVP (R6-Q4).

**Decision:** upload + forwarding address. **ASSUMED (self-answered) — review**

---

❓ **Q5** - **Setup data**: How do the menu, recipes and first prices get in?
Today they are three CSVs (`menu.csv`, `recipes.csv`, `price_history.csv`).

➡️ CSV import with the same columns as today (plus the unit fields from
R3-Q3), and in-app editing afterwards. Price history is optional at import:
it fills up from accepted invoices. Onboarding is done with the customer
(R6-Q3).

**Decision:** CSV import + in-app editing. **ASSUMED (self-answered) — review**

---

❓ **Q6** - **Notifications**: How do people learn about flags? In-app,
email, SMS, push?

➡️ In-app on the invoice page, plus one email digest per venue per day, sent
only when there is something flagged. No SMS or push.

**Decision:** in-app + daily digest when flagged. **ASSUMED (self-answered) — review**

---

## Round 3: the data model

Unblocked by R2-Q2 (tenancy), R2-Q3 (loop) and R2-Q5 (setup).

❓ **Q1** - **Product identity**: F3: matching is an exact string. Suppliers
print their own names (a hypothetical second supplier might print "HERNÖ GIN
70 CL 40,5%" for the demo's "Herno Gin 70cl"), and F4 shows two suppliers
already sell the same products. Do we keep exact names, fuzzy-match
automatically, or map each supplier's printed name to a venue's own product?

➡️ Each venue has its own **Products** (what recipes use). Each supplier's
printed line becomes a **Supplier Article** (supplier + printed name + unit)
that is mapped to one product, with a pack size. The mapping is proposed
(exact, then normalised name match) and confirmed at review, then reused for
every later invoice from that supplier. ADR 0003.

**Decision:** Product ← Supplier Article mapping, learned at review. **ASSUMED (self-answered) — review**

---

❓ **Q2** - **Which last price does a price move compare against?** F4: today
it is the product's last price from any supplier. "Overcharged" means "more
than this supplier charged me last time"; "should I switch" is a
cross-supplier question.

➡️ A **Price Move** compares against the last accepted price of the same
supplier article. If the product has no history with that supplier, the line
shows the latest price from another supplier as context, marked as such, and
is not flagged. ADR 0003.

**Decision:** same-supplier comparison; cross-supplier shown, not flagged. **ASSUMED (self-answered) — review**

---

❓ **Q3** - **Units and pack sizes**: F5: recipe quantities are fractions of
whatever unit the supplier sells. That breaks as soon as a supplier switches
from 70 cl to 1 l bottles.

➡️ Each product has a **base unit** in one dimension: volume (cl), mass (g) or
count (each). A supplier article states its pack size in the base unit (a
70 cl bottle = 70 cl; a 30 l keg = 3000 cl). Recipes use the base unit
(4 cl gin). Unit price per base unit is derived. No conversion across
dimensions (no densities).

**Decision:** base units + pack sizes. **ASSUMED (self-answered) — review**

---

❓ **Q4** - **Cost basis**: F6: the newest price is used regardless of the
invoice date. If a March invoice is uploaded in May, should it change today's
cost?

➡️ The **Current Cost** of a product is the price on the accepted invoice with
the latest invoice date (not upload date), across suppliers. A backdated
invoice adds history and can be flagged, but does not replace a newer price.

**Decision:** latest by invoice date. **ASSUMED (self-answered) — review**

---

❓ **Q5** - **VAT**: F2: costs and menu prices are ex VAT; nothing stores VAT.
Venues think in prices incl. VAT on the menu; Swedish VAT differs for food and
alcohol.

➡️ All margins are computed ex VAT, as today. Invoice VAT is extracted and
stored per invoice only to reconcile the total ("Att betala"). Menu prices are
stored ex VAT; the UI accepts a price incl. VAT and converts with a VAT rate
set per menu item (no rates are hard-coded in the product).

**Decision:** ex-VAT margins; VAT stored for reconciliation; per-item rate. **ASSUMED (self-answered) — review**

---

❓ **Q6** - **Line kinds**: F10: every row is treated as a product. Swedish
drinks invoices have pant (deposits), delivery fees, discounts, öresutjämning
(rounding to whole kronor), and credit notes.

➡️ Every **Invoice Line** has a kind: product, deposit, fee, discount,
rounding. Only product lines create price points; discounts printed on the
line are part of its net unit price; separate discount lines and fees are kept
but not spread over products. A **Credit Note** is its own document linked to
the invoice it corrects; it never creates price points.

**Decision:** five line kinds; credit notes separate. **ASSUMED (self-answered) — review**

---

❓ **Q7** - **Duplicates**: F7: nothing stops the same invoice being counted
twice (upload, then the forwarded email).

➡️ An invoice is unique per venue by (supplier, invoice number); a second copy
is refused with a link to the first. Identical files (same hash) are refused
before extraction, so they cost no LLM call.

**Decision:** as recommended. **ASSUMED (self-answered) — review**

---

❓ **Q8** - **Invoice lifecycle**: Which states?

➡️ **Received** (file stored) → **Draft** (extracted, awaiting review) →
**Accepted** or **Rejected**. If extraction fails: **Needs manual entry**
(the file beside an empty form). Accepted invoices can be reopened by an
Owner, which withdraws their price points until re-accepted.

**Decision:** as recommended. **ASSUMED (self-answered) — review**

---

## Round 4: pricing and margin logic, extraction, trust

Unblocked by R3 (data model) and R2-Q3 (loop).

❓ **Q1** - **Price-move threshold**: F8: a fixed 5%. Should it be
configurable, and per what?

➡️ Default 5% in either direction, configurable per venue and overridable per
product. A **Price Alert** is a price move at or over the threshold.

**Decision:** 5% default, per venue, per-product override. **ASSUMED (self-answered) — review**

---

❓ **Q2** - **Margin definition**: Today margin = 1 − serving cost / menu price
ex VAT, against a per-item target (F8). Keep it? Add SEK?

➡️ Keep the **Gross Margin** percentage and its per-item **Target Margin**.
Also show the SEK per serving lost or gained. Category defaults for targets
(drinks, food) are set at onboarding and copied onto items; the item value is
what counts.

**Decision:** as recommended. **ASSUMED (self-answered) — review**

---

❓ **Q3** - **Arithmetic check tolerance**: F9: exact to the öre. Real invoices
round the total to whole kronor.

➡️ Keep exact at line level and for the sum of lines. An explicit rounding
line (R3-Q6) is allowed to make up the difference to the printed total, up to
±1.00 SEK. Anything else is an **Arithmetic Issue**.

**Decision:** exact; rounding line ≤ ±1.00 SEK allowed. **ASSUMED (self-answered) — review**

---

❓ **Q4** - **Extraction engine for customers**: F11: today the local
`claude -p` CLI on the developer's login. That cannot serve customers. Options:
(a) Anthropic API server-side with a JSON schema; (b) a classic OCR/invoice
API with templates; (c) manual entry only.

➡️ (a). Server-side Anthropic API with structured output, the same rules (copy
numbers, never correct; ignore instructions in the file) and Go-side
validation, extended with line kind and VAT. The model is chosen by an
evaluation on at least 50 real pilot invoices, picking the cheapest model that
meets the accuracy bar (R5-Q5). The rule-based extractor stays for tests.
ADR 0002.

**Decision:** Anthropic API, model chosen by eval. **ASSUMED (self-answered) — review**

---

❓ **Q5** - **Auto-accept**: Should high-confidence invoices be accepted
without a person?

➡️ No. In the MVP every invoice needs a person's Accept; issues, unmatched
lines and alerts are highlighted to make that fast. Auto-accept is reconsidered
after the pilot with accuracy data. ADR 0004.

**Decision:** human Accept always. **ASSUMED (self-answered) — review**

---

❓ **Q6** - **Keep the originals?**

➡️ Yes, every received file is kept with its invoice for review and supplier
disputes, for as long as the customer stays (plus 30 days after cancellation).
We are not the venue's legal bookkeeping archive (R6-Q4).

**Decision:** keep originals; not the legal archive. **ASSUMED (self-answered) — review**

---

## Round 5: storage, auth, deployment, LLM operations

Unblocked by R1-Q4 (web app), R2-Q2 (tenancy), R3 (model) and R4-Q4
(extraction).

❓ **Q1** - **Database**: Managed Postgres, or SQLite in one file on one
server with continuous backup?

➡️ SQLite (WAL mode) inside the single Go process, continuously replicated to
EU object storage (Litestream or equivalent), with a tested restore. Move to
Postgres when a second app instance is needed. ADR 0005.

**Decision:** SQLite + continuous backup. **ASSUMED (self-answered) — review**

---

❓ **Q2** - **File storage**: Where do invoice files live?

➡️ S3-compatible object storage in an EU region, private bucket, one prefix
per venue, served only through the app.

**Decision:** EU object storage. **ASSUMED (self-answered) — review**

---

❓ **Q3** - **Auth**: Passwords, magic links, Google/Microsoft sign-in, BankID?

➡️ Email magic links and an HTTP-only session cookie; Owners invite Staff by
email. No passwords, no SSO, no BankID in the MVP.

**Decision:** magic links. **ASSUMED (self-answered) — review**

---

❓ **Q4** - **Deployment and UI stack**: Where and how does it run?

➡️ One Go binary serving server-rendered HTML (`html/template`, minimal
JavaScript) and the inbound-email webhook, in a container on one EU-hosted VM
or EU-region platform, TLS at the edge. Data stays in the EU (GDPR). One
staging and one production environment.

**Decision:** one binary, EU region, server-rendered UI. **ASSUMED (self-answered) — review**

---

❓ **Q5** - **LLM cost and reliability**: F11 measured USD 0.02–0.03 and
~7 s per invoice on a top-tier model via the CLI. What budget, retries and
fallbacks?

➡️ Extraction runs in a background job, never in the request. Timeout 60 s,
at most 2 attempts per invoice. Output must pass schema, field validation and
the duplicate check; arithmetic issues do not count as failure (they are the
product). On failure the invoice goes to Needs manual entry. Every call logs
model, tokens, cost and latency per venue. Budget: at most USD 0.05 per invoice
on average; alert if a venue's monthly LLM cost exceeds 10% of its
subscription. Accuracy bar for choosing the model: ≥ 98% of product lines with
qty, unit price and total exactly right on the eval set.

**Decision:** as recommended. **ASSUMED (self-answered) — review**

---

❓ **Q6** - **Operations**: What do we watch?

➡️ Structured `log/slog` JSON logs, an uptime check, an error alert by email,
a daily backup-restore check, and a weekly report of extraction accuracy
(edits made at review per line).

**Decision:** as recommended. **ASSUMED (self-answered) — review**

---

## Round 6: commercial model and scope

Unblocked by R1 (customer, market), R4-Q4/R5-Q5 (unit cost) and R5 (platform).

❓ **Q1** - **Pricing model**: Per invoice, per venue per month, tiers by
volume, or a share of savings?

➡️ A flat monthly price per venue in SEK, including a fair-use invoice volume
(e.g. 150 invoices a month), first two months free for pilots, annual plan
at a discount. Starting point to test in the pilot: 495 SEK per venue per
month ex VAT. Per-invoice pricing is rejected because it punishes the
behaviour we want (uploading every invoice).

**Decision:** flat per-venue subscription; price to be tested. **ASSUMED (self-answered) — review**

---

❓ **Q2** - **Payments**: How do customers pay?

➡️ Card via a hosted subscription billing provider that supports SEK (e.g.
Stripe Billing), with its customer portal for invoices and cancellation. No
self-built billing.

**Decision:** hosted subscription billing. **ASSUMED (self-answered) — review**

---

❓ **Q3** - **Onboarding**: Self-serve or done with the customer?

➡️ Done with the customer for the pilot and first paying venues: a 60-minute
session that imports their menu and recipes from whatever spreadsheet they
have and processes their last month of invoices. Self-serve signup exists but
is not the focus.

**Decision:** concierge onboarding. **ASSUMED (self-answered) — review**

---

❓ **Q4** - **Explicitly out of scope** for the MVP: is this the list?

➡️ Yes: POS integration and sales mix; inventory and stock counts; purchasing
and ordering; accounting/bookkeeping export (Fortnox, Visma) — first candidate
after the MVP; being the legal bookkeeping archive; Peppol/EDI e-invoices;
auto-accept; multi-currency and non-Swedish VAT; native mobile apps; recipe
yields, waste and prep sub-recipes; labour cost; supplier price lists and
supplier portals; an accountant role. ADR 0006 records the boundary.

**Decision:** as listed. **ASSUMED (self-answered) — review**

---

❓ **Q5** - **Legal minimum**: What must exist before taking money?

➡️ Terms of service, privacy policy, a data processing agreement listing the
sub-processors (hosting, object storage, email, LLM provider, billing), and a
data export (R7-Q2). Legal text to be reviewed by a lawyer, not written by us.

**Decision:** as recommended. **ASSUMED (self-answered) — review**

---

## Round 7: what the earlier rounds left open

Unblocked by R4-Q3 (arithmetic issues), R6-Q4 (scope) and R6-Q5 (legal).

❓ **Q1** - **Supplier disputes**: An arithmetic issue is only worth money if
the venue claims it back. Do we send emails to suppliers?

➡️ No sending. The invoice page offers a ready-to-copy dispute text (invoice
number, line, what was invoiced, what it should be) and lets the user mark the
issue Disputed or Resolved, so the digest stops repeating it.

**Decision:** copyable dispute text + status. **ASSUMED (self-answered) — review**

---

❓ **Q2** - **Data export**: Can customers take their data with them?

➡️ Yes: CSV export of products, supplier articles, price points, recipes, menu
and invoices, plus a zip of original files. Same columns as the import.

**Decision:** CSV + files export. **ASSUMED (self-answered) — review**

---

❓ **Q3** - **Who maintains product mappings when a supplier renames an
article?**

➡️ An unmatched line at review proposes the closest existing product; the
reviewer confirms or creates a new product. Old mappings are kept, so history
continues across the rename.

**Decision:** as recommended. **ASSUMED (self-answered) — review**

---

## Frontier

Empty. Every branch above has a decision; none depends on an open question.
What remains are the facts the pilot must produce (accuracy per model, the
price venues will pay, real invoice variety), which `docs/MVP_PLAN.md` lists
as risks and acceptance criteria, not as decisions.
