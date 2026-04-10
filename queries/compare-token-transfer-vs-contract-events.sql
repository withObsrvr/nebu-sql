with tt as (
  select count(*) as n
  from nebu('token-transfer', start = 60200000, stop = 60200000)
),
ce as (
  select count(*) as n
  from nebu('contract-events', start = 60200000, stop = 60200000)
)
select tt.n as token_transfer_rows, ce.n as contract_event_rows
from tt, ce;
