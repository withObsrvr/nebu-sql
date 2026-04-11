# Obsrvr Architecture / Product Map

## Summary

Obsrvr is a Stellar-native analytics stack with three core layers:

- **`nebu`** extracts and transforms Stellar ledger data through processors
- **`nebu-sql`** provides interactive SQL over processor outputs
- **`Obsrvr Lake`** materializes bronze, silver, and gold datasets for APIs, dashboards, and tenant-specific analytics

Together, these components form a coherent path from raw ledger events to user-facing insights.

## Executive Narrative

Obsrvr did not emerge as a single system all at once. It evolved through several generations of the same core idea: make Stellar data easy to extract, transform, query, and turn into products. The earliest generation, `cdp-pipeline-workflow`, proved the platform could run real production pipelines with many sources, processors, and sinks. `flowctl` extended that idea into a managed orchestration model suitable for user-defined pipelines and a Goldsky-like control plane. `nebu` then simplified the extraction layer into a much lighter, Unix-style processor runtime with a cleaner developer experience, while `nebu-sql` added an interactive SQL workbench for exploration and validation. In parallel, Obsrvr Lake became the durable medallion backend for bronze, silver, and gold datasets.

The result is not a set of competing tools, but a layered platform. `nebu` is the developer-facing extraction surface. `flowctl` is the orchestration engine for managed pipelines. Obsrvr Lake is the historical warehouse and semantic model layer. APIs, dashboards, and tenant-specific gold outputs are the delivery layer that turns data infrastructure into user-facing insight products.

---

## 1. Project Lineage

The current architecture is best understood as the result of several generations of the same core product thesis:

> Make Stellar data programmable, composable, queryable, and productizable.

### Generation 1 — `cdp-pipeline-workflow`
This was the original production workhorse and remains in use on the platform today.

**Primary strengths**
- broad source adapter support
- broad processor library
- broad consumer / sink library
- production-proven execution
- ability to support many use cases in one system

**Primary tradeoffs**
- large surface area
- heavier mental model
- more monolithic feel
- harder to present as a simple developer-facing product

**Best interpretation**
`cdp-pipeline-workflow` is the original production pipeline engine and the main source of proven platform logic, source adapters, and sink integrations.

### Generation 2 — `flowctl`
This was the move from a powerful pipeline engine toward a true user-facing ETL control plane.

**Primary strengths**
- orchestrator model
- embedded control plane
- component registration and discovery
- health monitoring and heartbeats
- DAG execution and richer topologies
- strong fit for managed pipelines and hosted execution

**Primary tradeoffs**
- heavier operational model
- more platform machinery than many users need
- less immediate than simple CLI-first processing

**Best interpretation**
`flowctl` is the orchestration and control-plane layer for managed, multi-component, user-defined pipelines.

### Generation 3 — `nebu`
This was the simplification step: keep the useful extraction ideas, but make the interface dramatically lighter.

**Primary strengths**
- tiny conceptual surface area
- Unix-style composition
- fast iteration loops
- standalone binaries
- stable processor contract
- easy onboarding for developers and agents

**Primary tradeoffs**
- not a full orchestration platform by itself
- not sufficient alone for managed multi-tenant pipeline products
- needs warehouse and delivery layers above it for full analytics products

**Best interpretation**
`nebu` is the canonical developer-facing extraction runtime and processor substrate.

### Generation 4 — Obsrvr Lake
This is the durable analytics substrate where medallion modeling, historical storage, and productizable datasets live.

**Primary strengths**
- bronze / silver / gold architecture
- historical storage and replayability
- semantic modeling
- query API and dashboard potential
- support for tenant-specific gold outputs

**Best interpretation**
Obsrvr Lake is the canonical data lakehouse and analytics model layer.

### Why this lineage matters
These systems should not be viewed as disconnected or redundant. They represent successive attempts to solve different parts of the same problem:

- `cdp-pipeline-workflow` optimized for getting pipelines working in production
- `flowctl` optimized for orchestrating managed pipelines as a platform
- `nebu` optimized for simplicity, extraction ergonomics, and composability
- Obsrvr Lake optimized for durable analytics, semantic layers, and product delivery

This architecture doc treats that overlap as intentional lineage, not accidental duplication.

---

## 2. Product Surfaces

### A. Extraction Surface — `nebu`

**Audience**
- developers
- protocol teams
- infrastructure operators
- agentic tooling

**Responsibilities**
- fetch ledger data
- run processors
- chain transforms and sinks
- emit JSON/protobuf event streams
- expose machine-readable schemas via `--describe-json`

**Examples**
- `token-transfer`
- `contract-events`
- `contract-invocation`
- `amount-filter`
- `dedup`
- `json-file-sink`
- `postgres-sink`

**Best for**
- live pipelines
- custom extraction
- narrow backfills
- prototyping event logic

---

### B. Interactive Analysis Surface — `nebu-sql`

**Audience**
- analysts
- data engineers
- developers exploring schemas
- agents writing SQL

**Responsibilities**
- expose processors as SQL table sources
- support ad hoc joins and aggregations
- provide reusable query recipes
- enable export to Parquet / JSON / CSV

**Example**

```sql
select *
from nebu('token-transfer', start = 60200000, stop = 60200010)
limit 10;
```

**Best for**
- exploratory analysis
- validating business logic before materialization
- cookbook queries
- lightweight local analytics

---

### C. Durable Analytics Surface — Obsrvr Lake

**Audience**
- product teams
- analytics consumers
- enterprise customers
- BI tools
- downstream services

**Responsibilities**
- maintain historical canonical data
- normalize data into reusable semantic tables
- manage medallion models
- power APIs and dashboards
- support tenant-specific gold layers

**Best for**
- repeatable analytics
- historical trend analysis
- production BI
- external-facing data products

---

### D. Delivery Surface — APIs / Dashboards / Gold Packages

**Audience**
- non-technical users
- customers
- internal stakeholders
- ecosystem teams

**Responsibilities**
- answer business questions
- expose curated metrics
- serve dashboards
- trigger alerts and webhooks
- support tenant-specific outputs

This is the user-facing insight layer.

---

## 2. Technical Architecture Map

### Layer 1 — Protocol Ingestion and Processor Runtime
**Owned by:** `nebu`

**Responsibilities**
- ledger access
- processor runtime contract
- extraction of domain events
- composable transforms and sinks
- schema discovery through `--describe-json`

**Core outputs**
- JSONL event streams
- processor-specific envelopes
- schemas for downstream consumers

**Example event families**
- token transfers
- contract events
- contract invocations
- transaction stats
- ledger change stats
- account effects

**Role in the stack**
This is the programmable edge of the system.

It answers:
- How do we get the right raw facts out of Stellar?
- How do we build a custom processor?
- How do we stream live events into another system?

---

### Layer 2 — Interactive SQL Workbench
**Owned by:** `nebu-sql`

**Responsibilities**
- expose processors as table functions
- support schema-aware SQL over processor outputs
- enable ad hoc joins and aggregates
- store reusable query files
- provide an agent-friendly query surface

**Example use cases**
- inspect row shape
- compare token-transfer vs contract-events
- prototype stablecoin volume logic
- validate contract-event taxonomy
- export ad hoc query results

**Role in the stack**
This is the analytics prototyping layer.

It answers:
- What is in this data?
- Can this metric be expressed correctly in SQL?
- Before promoting this logic into silver/gold, does it work?

---

### Layer 3 — Bronze Medallion Layer
**Owned by:** Obsrvr Lake

**Representative components**
- `stellar-postgres-ingester`
- `postgres-ducklake-flusher`
- `stellar-history-loader`

**Responsibilities**
- canonical ingestion of raw ledger-derived tables
- hot/cold storage
- Parquet / DuckLake durability
- source-of-truth low-level facts
- replayability and traceability

**Typical bronze tables**
- `ledgers_row_v2`
- `transactions_row_v2`
- `operations_row_v2`
- `effects_row_v1`
- `trades_row_v1`
- `contract_creations_v1`
- raw contract event streams
- raw state / account / trustline / offer facts

**Characteristics**
- minimally transformed
- close to source truth
- append-heavy
- replayable
- partitionable
- lineage-friendly

**Role in the stack**
This is the canonical raw data substrate.

It answers:
- What actually happened on-chain?
- What is the authoritative historical record?
- Can downstream models be rebuilt from scratch?

---

### Layer 4 — Silver Medallion Layer
**Owned by:** Obsrvr Lake

**Representative components**
- `silver-realtime-transformer`
- `silver-cold-flusher`

**Responsibilities**
- normalize bronze into domain-friendly tables
- reduce schema complexity
- parse and standardize protocol semantics
- support both realtime and backfill paths
- attach metadata and lineage

**Recommended canonical silver domains**

#### Asset and token domain
- `silver.token_transfers`
- `silver.token_mints`
- `silver.token_burns`
- `silver.token_clawbacks`
- `silver.asset_registry`
- `silver.asset_holders_snapshot`

#### Contract and Soroban domain
- `silver.contract_events`
- `silver.contract_invocations`
- `silver.contract_registry`
- `silver.contract_token_interactions`

#### Account and identity domain
- `silver.accounts_current`
- `silver.trustlines_current`
- `silver.account_activity_daily`
- `silver.address_labels`

#### Network domain
- `silver.ledger_metrics`
- `silver.transaction_metrics`
- `silver.operation_metrics`
- `silver.effects_normalized`

#### Metadata / support domain
- `silver.entity_registry`
- `silver.price_reference` (optional future)
- `_meta` lineage conventions across tables

**Characteristics**
- query-friendly
- semantically meaningful
- normalized enough for analytics
- reusable across multiple gold products

**Role in the stack**
This is the shared semantic model layer.

It answers:
- What are the canonical tables people should use?
- How do we hide raw ledger complexity?
- How do we avoid every dashboard reimplementing parsing logic?

---

### Layer 5 — Gold Medallion Layer
**Owned by:** Obsrvr Lake + product packages

**Responsibilities**
- package silver into use-case-specific metrics
- define business-ready models
- power dashboards and APIs
- support customer-specific transformations
- create reusable insight packs

**Recommended gold domains**

#### Gold: Stablecoin Intelligence
- `gold.stablecoin_daily_metrics`
- `gold.stablecoin_hourly_metrics`
- `gold.stablecoin_issuer_activity`
- `gold.stablecoin_top_counterparties`
- `gold.stablecoin_large_transfers`
- `gold.stablecoin_contract_usage`
- `gold.stablecoin_flow_concentration`

#### Gold: Soroban / Protocol Activity
- `gold.protocol_daily_activity`
- `gold.top_contracts_by_users`
- `gold.contract_event_taxonomy`
- `gold.contract_token_flow_summary`

#### Gold: Network Health
- `gold.network_daily_health`
- `gold.fee_trends`
- `gold.tx_mix`
- `gold.ledger_throughput`
- `gold.soroban_share_of_activity`

#### Gold: Onboarding / C-address Tooling
- `gold.c_address_funding_flows`
- `gold.deposit_forwarding_status`
- `gold.bridge_to_contract_success_rates`
- `gold.asset_funding_paths`

**Role in the stack**
This is the insight product layer.

It answers:
- What should the user actually look at?
- Which metrics should the API expose?
- Which dashboard sections exist?
- What products are we publishing or selling?

---

## 3. What Belongs Where

### What stays in `nebu`
Put logic here if it is:
- event extraction logic
- reusable chain processing primitive
- generic transform
- sink
- CLI-oriented processor
- intended to stream or pipe

**Examples**
- token transfer extraction
- contract event extraction
- invocation extraction
- generic amount filter
- dedup
- file or NATS sinks

**Do not put here**
- tenant-specific business metrics
- complex historical materializations
- dashboards
- labeled entity analytics
- long-lived aggregate tables

---

### What belongs in `nebu-sql`
Put logic here if it is:
- exploratory SQL
- a reference query
- a cookbook pattern
- logic under validation before silver/gold promotion
- a developer or analyst recipe

**Examples**
- top assets in a ledger range
- compare transfer vs contract event counts
- join transaction hashes across processors
- inspect event-type mix

**Do not treat as final product surface for**
- canonical production metrics tables
- deeply curated historical models
- customer-facing APIs
- repeated heavy long-range computation

`nebu-sql` is the workbench, not the warehouse.

---

### What belongs in Bronze
Put data here if it is:
- source-aligned
- low-level
- append-only
- replayable
- minimally transformed
- the raw chain fact substrate

**Examples**
- raw ledger rows
- raw transaction rows
- raw operations and effects
- raw contract events
- raw token transfer extracts stored as facts

---

### What belongs in Silver
Put data here if it is:
- a reusable semantic model
- shared across multiple use cases
- normalized and documented
- stable enough to support several gold products

**Examples**
- canonical `token_transfers`
- normalized `contract_events`
- `contract_registry`
- `asset_registry`
- account and trustline state
- address or entity labels

---

### What belongs in Gold
Put data here if it is:
- use-case specific
- dashboard-ready
- business-friendly
- aggregated
- tenant-specific
- tied directly to an insight product

**Examples**
- daily stablecoin volumes
- issuer monitoring
- top counterparties
- large transfer alerts
- protocol adoption metrics
- asset onboarding metrics

---

## 4. End-to-End Data Flow

### A. Exploration Flow

```text
Stellar ledgers
  -> nebu processor
  -> nebu-sql table function
  -> SQL exploration / query files / parquet export
```

Use this for:
- prototyping
- debugging
- quick insight generation
- validating new processor designs

---

### B. Production Analytics Flow

```text
Stellar ledgers
  -> bronze ingestion
  -> bronze hot/cold
  -> silver realtime/cold transforms
  -> gold materializations
  -> query API / dashboards / customer outputs
```

Use this for:
- historical analytics
- repeated metric computation
- customer-facing products
- data APIs

---

### C. Product Development Flow

```text
1. Prototype extraction in nebu
2. Test metric logic in nebu-sql
3. Promote stable logic into silver/gold models
4. Expose via query API/dashboard
5. Stream tenant-specific gold outputs where needed
```

This is the intended lifecycle for new analytics products.

---

## 5. Flagship Product Map: Stablecoin Intelligence

### Product name
**Obsrvr Stablecoin Intelligence for Stellar**

### Why start here
- Stellar has real stablecoin relevance
- existing processors already expose much of the needed signal
- SQL analytics maps naturally to the use case
- it is easy to explain to outsiders
- it is useful for issuers, wallets, protocols, and ecosystem teams

### Extraction layer
Using `nebu`:
- `token-transfer`
- `contract-events`
- optionally `contract-invocation`
- later, issuer / balance-specific processors as needed

### SQL workbench
Using `nebu-sql`:
- validate transfer volume logic
- validate mint / burn / clawback logic
- join contract events to token transfers
- prototype active address metrics

### Recommended silver tables

#### `silver.token_transfers`
- `ledger_sequence`
- `tx_hash`
- `from_address`
- `to_address`
- `asset_code`
- `asset_issuer`
- `amount`
- `transfer_type`
- `contract_id`
- `close_time`
- `_meta`

#### `silver.assets`
- `asset_id`
- `asset_code`
- `asset_issuer`
- `decimals`
- `is_stablecoin`
- `label`
- `metadata_source`

#### `silver.contract_events`
- `tx_hash`
- `contract_id`
- `event_type`
- `event_subtype`
- `payload_json`
- `close_time`

#### `silver.address_labels`
- `address`
- `entity_name`
- `entity_type`
- `confidence`
- `source`

#### `silver.issuer_actions`
- `issuer`
- `asset`
- `action_type`
- `amount`
- `tx_hash`
- `close_time`

### Recommended gold tables
- `gold.stablecoin_daily_metrics`
- `gold.stablecoin_hourly_metrics`
- `gold.stablecoin_top_counterparties`
- `gold.stablecoin_large_transfers`
- `gold.stablecoin_issuer_monitoring`
- `gold.stablecoin_contract_usage`
- `gold.stablecoin_flow_concentration`

### Delivery surfaces
Expose via:
- dashboards
- query API endpoints
- downloadable Parquet / CSV
- saved SQL packs
- customer-specific gold streams where needed

---

## 6. Recommended First Gold Packages

### Package 1: Stablecoin Intelligence
**Questions answered**
- Which stablecoins are most active?
- What is daily transfer volume?
- Who is minting, burning, or clawing back?
- Which counterparties dominate flows?
- Which contracts and protocols drive usage?

**Primary users**
- issuers
- protocols
- ecosystem analysts
- wallets
- compliance and risk teams

---

### Package 2: Soroban Activity Intelligence
**Questions answered**
- Which contracts are most active?
- Which event types dominate?
- Which protocols are growing?
- Which token flows are tied to Soroban apps?
- How is contract usage changing over time?

**Primary users**
- protocol teams
- ecosystem growth teams
- infrastructure providers
- app builders

---

### Package 3: Network Health Intelligence
**Questions answered**
- How many transactions, operations, and effects occur per day?
- What is Soroban's share of activity?
- How are fees changing?
- Which ledgers show event spikes?
- What structural network changes are occurring over time?

**Primary users**
- ecosystem operations teams
- researchers
- validators and infrastructure teams
- public dashboard users

---

## 7. APIs and Output Products

### For developers
- `nebu` CLI
- `nebu-sql`
- schema docs
- query packs
- Parquet exports

### For analysts
- SQL query library
- DuckDB starter notebooks
- gold model definitions
- API docs

### For business users
- dashboards
- CSV downloads
- scheduled reports
- alerting and webhooks

### For enterprise / tenants
- dedicated gold pipelines
- custom sinks
- tenant-isolated APIs
- contract-specific or entity-specific views

---

## 8. Role of `flowctl` vs `nebu` vs `cdp-pipeline-workflow`

This section clarifies how the major systems should relate going forward.

### `nebu`
**Strategic role:** canonical processor SDK / runtime

Use `nebu` when the job is:
- extracting facts from Stellar
- building standalone processors
- composing shell-friendly transforms
- supporting local analytics and agent workflows
- defining stable processor contracts and schemas

**Build new here when**
- the logic is a clean extraction primitive
- the output has a stable schema
- the processor is useful standalone
- stdin/stdout composition is a good fit
- the functionality should be broadly reusable by developers

**Examples**
- token transfer extraction
- contract event extraction
- issuer action extraction
- contract classification extractors
- bridge or deposit monitoring primitives

---

### `flowctl`
**Strategic role:** orchestration and control plane for managed pipelines

Use `flowctl` when the job is:
- running hosted or user-defined pipelines
- coordinating multi-component DAGs
- handling registration, health checks, retries, and lifecycle management
- supporting tenant-managed execution
- acting as the backend for a pipeline UI or platform experience

**Build new here when**
- orchestration is the primary need
- multi-stage DAG execution is required
- components need lifecycle management
- tenant isolation and managed execution matter
- the system is closer to a control plane than a simple CLI pipeline

**Examples**
- user-defined ETL pipeline execution
- tenant-specific gold transformation pipelines
- fan-out from shared silver data to custom sinks
- managed scheduled pipeline runs

---

### `cdp-pipeline-workflow`
**Strategic role:** legacy execution engine and migration source

Use `cdp-pipeline-workflow` when the job is:
- supporting current production workloads already built on it
- maintaining compatibility with existing platform behavior
- preserving proven source adapters, processors, and consumers while migration is incomplete

**Recommended posture**
- keep it working where it already powers the platform
- avoid expanding it as the long-term flagship abstraction unless necessary
- treat it as a source of proven logic to harvest into newer layers

**Extract from it over time**
- proven source adapters
- durable sink integrations
- domain-specific processor logic
- schema knowledge encoded in transforms
- successful silver/gold patterns worth preserving

---

### Obsrvr Lake
**Strategic role:** canonical analytics warehouse and semantic model layer

Use Obsrvr Lake when the job is:
- storing canonical historical facts
- materializing bronze/silver/gold datasets
- powering query APIs, dashboards, and reports
- supporting tenant-specific analytics outputs
- turning extracted facts into durable insight products

**Build new here when**
- the output is a reusable semantic table
- the logic should be historical and durable
- the result powers dashboards or APIs
- the system needs medallion discipline and lineage

---

### Recommended hierarchy
For users, the platform should not feel like four competing products. The clean hierarchy is:

1. **Obsrvr Platform** — the overall analytics and data-product experience
2. **`nebu`** — the developer-facing extraction and processor surface
3. **`flowctl`** — the managed orchestration backend for pipelines
4. **Obsrvr Lake** — the medallion warehouse and semantic analytics substrate
5. **`cdp-pipeline-workflow`** — the current legacy/production engine, gradually harvested and migrated

This allows the external story to stay simple while preserving the value of each internal system.

---

## 9. Canonical Repo Responsibility Map

### Repo: `nebu`
**Owns**
- processor contract
- processor CLI/runtime
- reference processors
- transforms and sinks
- extraction docs
- `--describe-json` protocol

**Potential additions**
- more extraction processors for missing business primitives
- example pipelines
- SDK/operator docs

---

### Repo: `nebu-sql`
**Owns**
- `nebu()` table function
- query runner
- cookbook
- reusable reference SQL
- exploration recipes

**Potential additions**
- `queries/stablecoins/*.sql`
- `queries/soroban/*.sql`
- `queries/network/*.sql`
- guidance on promoting logic from SQL prototypes to gold models

---

### Repo / system: `obsrvr-lake`
**Owns**
- bronze ingestion
- silver transforms
- gold materialization jobs
- lineage and metadata
- hot/cold storage
- query API
- tenant gold infrastructure

**Potential additions**
- explicit medallion model docs
- canonical schema docs
- gold package definitions
- dashboard and API contracts

---

## 10. Recommended Migration Strategy

The goal is not to force an immediate rewrite. The goal is to progressively align each system to its strongest long-term role.

### Migration principles
- preserve production stability first
- prefer extraction of reusable primitives over wholesale rewrites
- move new development toward the intended long-term home
- keep user-facing product language simpler than the internal implementation reality

### From `cdp-pipeline-workflow` to newer layers

**Preserve and harvest**
- reusable source adapters
- sink integrations with real operational value
- domain-specific processor logic that has proven correctness
- schema and transformation knowledge already encoded in production paths

**Avoid carrying forward unchanged**
- one-off consumers with narrow platform coupling
- ad hoc glue code that fights the simpler `nebu` model
- abstractions whose only value is historical inertia

**Preferred migration path**
- extraction primitives move into `nebu`
- orchestration concerns move into `flowctl`
- durable semantic models move into Obsrvr Lake
- legacy execution remains in place until replacements are production-safe

### From `flowctl` into the broader platform

`flowctl` should not need to be the first thing every user learns. Instead, it should become the managed execution substrate behind user-defined pipelines.

**Recommended role**
- backend orchestration engine for the Obsrvr platform
- control plane for hosted pipeline execution
- scheduler and lifecycle manager for tenant-specific flows

**Implication**
The flagship product should be the broader Obsrvr platform, not `flowctl` as a standalone identity.

### From `nebu` into the broader platform

`nebu` should become the canonical place for new extraction-oriented development.

**Recommended role**
- preferred home for new processors
- preferred home for stable extraction contracts
- developer-facing substrate for experimentation and local workflows
- workbench companion to `nebu-sql`

**Implication**
If a new idea is fundamentally “extract a clean Stellar fact stream,” it should usually start in `nebu`.

### From SQL prototypes to durable analytics

Use a standard promotion path:

1. prototype extraction in `nebu`
2. validate business logic in `nebu-sql`
3. materialize canonical reusable models in silver
4. aggregate into gold for a specific use case
5. expose via API, dashboard, report, or tenant feed

This turns ad hoc experiments into productizable analytics without skipping semantic discipline.

### Practical next-step posture
- keep `cdp-pipeline-workflow` stable for current platform workloads
- use `flowctl` where orchestration and managed multi-tenant execution are required
- bias new extraction and processor work toward `nebu`
- bias durable analytics and productized metrics toward Obsrvr Lake
- present the whole system externally as one Obsrvr platform

---

## 11. Build Workflow for New Insight Products

Use the following workflow for new analytics products:

1. **Extraction hypothesis**
   - Determine which chain facts are needed
   - Implement or reuse processors in `nebu`

2. **Interactive validation**
   - Prototype metric logic in `nebu-sql`
   - Confirm correctness and query ergonomics

3. **Semantic promotion**
   - Move reusable logic into silver tables/models
   - Standardize schemas and documentation

4. **Productization**
   - Move user-facing aggregates into gold
   - Define metrics, refresh strategy, and ownership

5. **Delivery**
   - Expose through dashboards, APIs, reports, Parquet, or tenant feeds

This keeps exploration, canonical modeling, and product delivery cleanly separated.

---

## 12. Concrete Example: Daily USDC Transfer Volume

### In `nebu`
Use the `token-transfer` processor to extract token transfer events.

### In `nebu-sql`
Prototype the metric:

```sql
select
  date_trunc('day', to_timestamp(cast(json_extract_string(meta, '$.closedAtUnix') as bigint))) as day,
  json_extract_string(transfer, '$.assetCode') as asset,
  json_extract_string(transfer, '$.assetIssuer') as issuer,
  sum(cast(json_extract_string(transfer, '$.amount') as bigint)) as volume
from nebu('token-transfer', start = 60200000, stop = 60300000)
where event_type = 'transfer'
  and json_extract_string(transfer, '$.assetCode') = 'USDC'
group by 1, 2, 3
order by 1;
```

### In Silver
Materialize canonical `silver.token_transfers`.

### In Gold
Build `gold.stablecoin_daily_metrics`.

### In Delivery
Expose through:
- a dashboard chart
- an API endpoint such as `/stablecoins/daily-volume?asset=USDC`
- downloadable CSV or Parquet
- tenant reports or alerts

This is the lifecycle every insight should follow.

---

## 13. Suggested Immediate Next Actions

1. **Publish a single cross-stack architecture narrative**
   - explain `nebu` = extract
   - `nebu-sql` = explore
   - Obsrvr Lake = materialize
   - Gold/API = deliver

2. **Define canonical silver schemas**
   - especially for `token_transfers`, `contract_events`, `contract_invocations`, `asset_registry`, and `address_labels`

3. **Create the first gold package**
   - Stablecoin Intelligence
   - include schema, metric definitions, API endpoints, and dashboard wireframes

4. **Expand `nebu-sql` query packs**
   - organize by use case rather than processor
   - e.g. `queries/stablecoins`, `queries/soroban`, `queries/network`

5. **Build one polished end-to-end demo**
   - live extraction in `nebu`
   - quick SQL in `nebu-sql`
   - the same logic materialized in Obsrvr Lake gold
   - surfaced in API or dashboard form

---

## 14. Simplified Mental Model

### `nebu`
Get facts out of Stellar.

### `nebu-sql`
Ask questions of extracted facts.

### Bronze
Store canonical facts.

### Silver
Standardize facts into reusable models.

### Gold
Turn models into products and insights.

### APIs / Dashboards / Tenant Outputs
Deliver answers to users.
