---
status: proposed
---

# SQLite in the single Go process, continuously backed up, until a second instance is needed

The MVP runs as one Go binary on one EU-hosted server and stores its data in SQLite (WAL mode) in that process, replicated continuously to EU object storage with a restore that is tested daily; invoice files live in object storage. We chose this over managed Postgres because the expected load (a few dozen venues, a few hundred invoices a day) fits one process with room to spare, and it removes a network hop, a second service to run and a monthly bill. The trigger to move to Postgres is needing more than one app instance (for availability or load); the schema keeps to portable SQL to make that move mechanical.

## Consequences

- Only one process may write: no horizontal scaling and a short downtime on deploys.
- A pure-Go SQLite driver is the one notable non-standard-library dependency.
