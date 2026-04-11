select
  json_extract_string(transfer, '$.assetCode') as asset,
  count(*) as n,
  sum(cast(json_extract_string(transfer, '$.amount') as bigint)) as total
from nebu('token-transfer', start = 60200000, stop = 60200100)
where event_type = 'transfer'
group by 1
order by total desc;
