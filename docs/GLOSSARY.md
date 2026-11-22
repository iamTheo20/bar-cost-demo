# Bar Cost

Watches what a bar or restaurant pays its suppliers and what that does to the
margin on each drink and dish, so the venue notices overcharges and margin
losses within a day of the invoice arriving.

## People and places

**Organisation**:
The business that subscribes and pays; it owns one or more venues.
_Avoid_: Account, customer, tenant, company

**Venue**:
One bar or restaurant with its own menu, recipes, suppliers and prices.
_Avoid_: Site, location, restaurant, store

**Owner**:
A member of an organisation who may do everything, including billing and inviting members.
_Avoid_: Admin, manager

**Staff**:
A member who handles invoices, products, recipes and the menu, but not billing or members.
_Avoid_: Employee, user, team member

**Supplier**:
A company that sells goods to a venue and sends it invoices.
_Avoid_: Vendor, wholesaler, distributor

## Buying

**Supplier Invoice**:
A supplier's demand for payment for goods delivered to a venue; "invoice" for short.
_Avoid_: Bill, receipt, faktura

**Credit Note**:
A supplier's document that reduces what a venue owes on an earlier invoice.
_Avoid_: Refund, negative invoice, kreditfaktura

**Invoice Line**:
One row of an invoice: a product line, a deposit, a fee, a discount or a rounding.
_Avoid_: Row, item, entry

**Deposit**:
An amount charged for returnable packaging, such as kegs, crates and bottles; it is not a cost of the goods.
_Avoid_: Pant (in English text), packaging fee

**Arithmetic Issue**:
A place where the supplier's own numbers on an invoice do not add up: quantity times unit price differs from the line total, or the lines differ from the subtotal.
_Avoid_: Error, discrepancy, mismatch

**Dispute**:
A venue's claim to a supplier that an arithmetic issue or a price is wrong.
_Avoid_: Complaint, claim, ticket

## Invoice lifecycle

**Received**:
An invoice whose file the venue has sent in, not yet read.
_Avoid_: Uploaded, pending

**Draft**:
An invoice that has been read and waits for a person to review it.
_Avoid_: Extracted, unconfirmed

**Review**:
A person checking a draft against the original, fixing what was misread and matching each product line to a product.
_Avoid_: Approval, verification

**Accepted**:
An invoice a person has confirmed; only accepted invoices count towards prices and margins.
_Avoid_: Approved, booked, posted, final

**Rejected**:
An invoice a person has decided not to count, such as a duplicate or a non-invoice.
_Avoid_: Deleted, discarded

**Needs Manual Entry**:
An invoice that could not be read automatically and waits for a person to type it in.
_Avoid_: Failed, error

## Products and prices

**Product**:
Something a venue buys and uses in recipes, named the venue's way.
_Avoid_: Ingredient (for the product itself), item, SKU, article

**Supplier Article**:
A product as one supplier sells and prints it: that supplier's name for it, its unit and its pack size.
_Avoid_: Supplier product, SKU, listing

**Base Unit**:
The single unit a product is measured in for recipes and costs: centilitres, grams or pieces.
_Avoid_: Unit (on its own), measure

**Pack Size**:
How many base units one supplier article contains, such as 70 cl for a bottle.
_Avoid_: Case size, package

**Price Point**:
The unit price a venue paid for a supplier article on one accepted invoice, on that invoice's date.
_Avoid_: Price record, history row

**Last Price**:
The most recent price point of the same supplier article before an invoice's date.
_Avoid_: Previous price, old price, reference price

**Price Move**:
The change from the last price to the price on a new invoice, as a percentage.
_Avoid_: Price change, delta, swing

**Price Alert**:
A price move at or over the venue's threshold, in either direction.
_Avoid_: Flag, warning, price jump

**Current Cost**:
A product's price per base unit on the accepted invoice with the latest invoice date, from any supplier.
_Avoid_: Current price, latest price, cost price

## Menu and margins

**Menu Item**:
A drink or dish the venue sells, with its price excluding VAT.
_Avoid_: Dish, drink, product, SKU

**Recipe**:
What one serving of a menu item uses: a list of products and quantities in base units.
_Avoid_: Spec, build, bill of materials

**Ingredient**:
One line of a recipe: a product and the quantity one serving uses.
_Avoid_: Component, recipe line

**Serving Cost**:
What the ingredients of one serving cost at current costs.
_Avoid_: Cost of goods, plate cost, pour cost

**Gross Margin**:
The share of a menu item's price excluding VAT left after its serving cost; "margin" for short.
_Avoid_: Markup, profit, GP

**Target Margin**:
The gross margin a venue wants a menu item to keep.
_Avoid_: Goal, minimum margin, threshold

**Below Target**:
A menu item whose gross margin at current costs is lower than its target margin.
_Avoid_: Unprofitable, losing money, red

**Affected Menu Item**:
A menu item whose recipe uses a product on the invoice being looked at.
_Avoid_: Impacted item

**Digest**:
The daily message to a venue listing new price alerts, below-target items and open arithmetic issues; sent only when there is something to list.
_Avoid_: Report, newsletter, notification
