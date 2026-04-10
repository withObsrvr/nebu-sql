# nebu-sql

`nebu-sql` makes installed nebu processors queryable as DuckDB table sources.

Instead of manually piping processor output into `read_json('/dev/stdin')`, you can write SQL like:

```sql
SELECT *
FROM nebu('token-transfer', start = 60200000, stop = 60200001)
LIMIT 5;
```

It is a small standalone binary that embeds DuckDB, discovers processor schemas through `--describe-json`, and streams JSONL output from processor binaries directly into SQL queries.

## Status

This project is still early, but the core path works today.

Current implementation:

- embeds DuckDB via `duckdb-go`
- registers a `nebu()` table function
- resolves installed processor binaries on `PATH`
- calls `--describe-json` to discover top-level columns
- streams JSONL rows from processor stdout into DuckDB

At the moment, processors must support `--describe-json`.

The query is also bound to the process context, so Ctrl-C / query cancellation tears down the spawned processor subprocess.

## v0 shape

- separate binary
- DuckDB embedded via Go
- `nebu()` table function
- `-c` and `--file` query modes
- explicit args only: processor, `start`, `stop`
- `_schema`, `_nebu_version`, and synthetic `event_type` columns are always present
- additional top-level columns are derived from the processor's `--describe-json` schema

Top-level scalar fields are exposed as strings. Nested objects and arrays are exposed as JSON-encoded `VARCHAR` values, so you can query them with DuckDB JSON functions. Absent payloads are currently emitted as empty strings, so use `event_type` to distinguish row shapes when relevant.

## Quick start

```bash
# install nebu-sql
go install github.com/withObsrvr/nebu-sql/cmd/nebu-sql@latest

# make sure the processor you want is installed
nebu install token-transfer

# run a query
nebu-sql -c "
  select count(*)
  from nebu('token-transfer', start = 60200000, stop = 60200001)
"
```

## Why this instead of just piping to DuckDB?

You can already do useful work with:

```bash
token-transfer --start-ledger 60200000 --end-ledger 60200100 | \
  duckdb -c "SELECT ... FROM read_json('/dev/stdin')"
```

That remains a great low-level primitive. `nebu-sql` exists for the cases where you want processors to be **first-class SQL inputs** rather than anonymous stdin streams.

Why that matters:

- **Less query boilerplate** — you write `FROM nebu('token-transfer', ...)` instead of rebuilding the same `read_json('/dev/stdin')` pattern every time.
- **Schema-aware discovery** — `nebu-sql` uses `--describe-json` to expose processor-specific top-level columns, so the query surface follows the processor contract.
- **One SQL surface for many processors** — every installed processor that supports `--describe-json` becomes queryable through the same function shape.
- **Better multi-processor workflows** — it becomes natural to compare or combine processor outputs in SQL without manually wiring separate shell pipelines for each query.
- **A cleaner foundation for tools and agents** — `nebu('processor', ...)` is a much better target for saved queries, notebooks, demos, and future agent-written SQL than ad hoc shell pipelines.

In short: piping to DuckDB is the Unix primitive; `nebu-sql` turns that primitive into a reusable SQL interface.

## Examples

```bash
go run ./cmd/nebu-sql -c "select count(*) from nebu('token-transfer', start = 60200000, stop = 60200001)"
```

```bash
go run ./cmd/nebu-sql -c "
  select
    json_extract_string(transfer, '$.assetCode') as asset,
    count(*) as n
  from nebu('token-transfer', start = 60200000, stop = 60200001)
  where event_type = 'transfer'
  group by 1
  order by 2 desc
"
```

```bash
go run ./cmd/nebu-sql --json -c "
  select contractId, eventType, type
  from nebu('contract-events', start = 60200000, stop = 60200000)
  limit 3
"
```

## Development

The official development environment for this repo is the Nix flake:

```bash
nix develop
```

That shell provides the pinned Go toolchain, DuckDB, GoReleaser, and the other tools used by the project. CI also verifies both `nix develop` and `nix build` so the flake stays healthy.

For local development, the fastest loop is:

1. install or update a processor with `nebu install <name>`
2. run `make test`
3. run a local query with `make run -- -c "..."`
4. use `make smoke-real` to check behavior against several real processors


```bash
# run tests
make test

# build the binary
make build

# print version
go run ./cmd/nebu-sql --version

# build with an injected version locally
go build -ldflags "-X github.com/withObsrvr/nebu-sql/internal/version.Value=v0.1.0" ./cmd/nebu-sql

# run locally
make run -- -c "select count(*) from nebu('token-transfer', start = 60200000, stop = 60200000)"

# smoke-test several real installed processors
make smoke-real
```

Current tests cover:

- schema extraction from `--describe-json`
- row normalization and event-type detection
- dynamic column construction
- cancellation classification
- end-to-end table-function execution with a fake processor binary
- a smoke-test script for several real processors on PATH (skipping ones without `--describe-json`)

## Release

- `--version` reads the injected linker value when present
- `.goreleaser.yaml` injects `github.com/withObsrvr/nebu-sql/internal/version.Value={{.Version}}`
- the Nix package version derives from flake metadata (`rev` / `shortRev` / `lastModifiedDate`) when available
- GitHub Actions CI runs on pushes and pull requests, including `nix develop` and `nix build`
- GitHub Actions release automation runs on `v*` tags
