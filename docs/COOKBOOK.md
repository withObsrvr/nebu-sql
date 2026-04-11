# nebu-sql Cookbook

Practical queries you can run immediately with `nebu-sql`.

These examples assume:

- `nebu-sql` is installed
- the referenced processor is installed via `nebu install <name>`
- you are querying a small ledger range first

A good default habit is to start with one ledger or a very small range, make sure the shape looks right, and then widen the range.

---

## 1. Count rows from a processor

```bash
nebu-sql -c "
  select count(*) as n
  from nebu('token-transfer', start = 60200000, stop = 60200000)
"
```

Use this first to sanity-check that a processor is working and returning rows.

---

## 2. See the row shape

```bash
nebu-sql -c "
  select *
  from nebu('token-transfer', start = 60200000, stop = 60200000)
  limit 3
"
```

This is the fastest way to inspect top-level columns and payload fields for a processor.

---

## 3. Count token-transfer event types

```bash
nebu-sql -c "
  select event_type, count(*) as n
  from nebu('token-transfer', start = 60200000, stop = 60200010)
  group by 1
  order by 2 desc
"
```

Useful for understanding the mix of `transfer`, `fee`, `mint`, `burn`, and `clawback` rows.

---

## 4. Top assets by transfer count

```bash
nebu-sql -c "
  select
    json_extract_string(transfer, '$.assetCode') as asset,
    count(*) as n
  from nebu('token-transfer', start = 60200000, stop = 60200100)
  where event_type = 'transfer'
  group by 1
  order by 2 desc
"
```

---

## 5. Top assets by transfer volume

```bash
nebu-sql -c "
  select
    json_extract_string(transfer, '$.assetCode') as asset,
    count(*) as transfers,
    sum(cast(json_extract_string(transfer, '$.amount') as bigint)) as total
  from nebu('token-transfer', start = 60200000, stop = 60200100)
  where event_type = 'transfer'
  group by 1
  order by total desc
"
```

---

## 6. Explore contract events

```bash
nebu-sql -c "
  select eventType, type, count(*) as n
  from nebu('contract-events', start = 60200000, stop = 60200010)
  group by 1, 2
  order by 3 desc
"
```

This is a good first query for seeing what kinds of contract events dominate a range.

---

## 7. Look at account-effects by effect type

```bash
nebu-sql -c "
  select type, count(*) as n
  from nebu('account-effects', start = 60200000, stop = 60200010)
  group by 1
  order by 2 desc
"
```

---

## 8. Inspect per-ledger summaries

### transaction-stats

```bash
nebu-sql -c "
  select *
  from nebu('transaction-stats', start = 60200000, stop = 60200010)
"
```

### ledger-change-stats

```bash
nebu-sql -c "
  select ledgerSequence, ledgerEntriesCreated, ledgerEntriesUpdated, ledgerEntriesDeleted
  from nebu('ledger-change-stats', start = 60200000, stop = 60200010)
  order by ledgerSequence
"
```

These processors are especially useful because they emit compact per-ledger summaries rather than one row per event.

---

## 9. Compare processor row counts

```bash
nebu-sql -c "
  with tt as (
    select count(*) as n
    from nebu('token-transfer', start = 60200000, stop = 60200000)
  ),
  ce as (
    select count(*) as n
    from nebu('contract-events', start = 60200000, stop = 60200000)
  )
  select tt.n as token_transfer_rows, ce.n as contract_event_rows
  from tt, ce
"
```

This is the beginning of differential validation: checking how different processors describe the same ledger range.

---

## 10. Join processor outputs on transaction hash

```bash
nebu-sql -c "
  with tt as (
    select
      json_extract_string(meta, '$.txHash') as tx_hash,
      json_extract_string(transfer, '$.assetCode') as asset
    from nebu('token-transfer', start = 60200000, stop = 60200010)
    where event_type = 'transfer'
  ),
  ce as (
    select
      transactionHash as tx_hash,
      eventType
    from nebu('contract-events', start = 60200000, stop = 60200010)
  )
  select ce.eventType, tt.asset, count(*) as n
  from ce
  join tt using (tx_hash)
  group by 1, 2
  order by 3 desc
"
```

This is one of the most compelling reasons to use `nebu-sql` instead of ad hoc shell pipes: joins across processor outputs become natural.

---

## 11. Export to Parquet

```bash
nebu-sql -c "
  copy (
    select *
    from nebu('token-transfer', start = 60200000, stop = 60200100)
    where event_type = 'transfer'
  ) to 'transfers.parquet' (format parquet)
"
```

DuckDB gives you export for free, so `nebu-sql` can be a lightweight bridge from processors to durable analytics files.

---

## 12. Emit JSON rows for scripts or agents

```bash
nebu-sql --json -c "
  select contractId, eventType, type
  from nebu('contract-events', start = 60200000, stop = 60200000)
  limit 5
"
```

This is useful when you want structured output for `jq`, shell scripts, or agents.

---

## 13. Save reusable query files

Create a file like `queries/top-assets.sql`:

```sql
select
  json_extract_string(transfer, '$.assetCode') as asset,
  count(*) as n,
  sum(cast(json_extract_string(transfer, '$.amount') as bigint)) as total
from nebu('token-transfer', start = 60200000, stop = 60200100)
where event_type = 'transfer'
group by 1
order by total desc;
```

Run it with:

```bash
nebu-sql --file queries/top-assets.sql
```

This is a good workflow for repeatable analyses, demos, and shared SQL recipes.

---

## Tips

- Start with a tiny ledger range before widening queries.
- Use `limit 1` or `limit 5` to inspect new processors.
- Use `event_type` when a processor has multiple row variants.
- Use `--json` when the output needs to feed another tool.
- If a processor fails, first check whether it supports `--describe-json`.
