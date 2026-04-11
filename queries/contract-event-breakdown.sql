select
  eventType,
  type,
  count(*) as n
from nebu('contract-events', start = 60200000, stop = 60200010)
group by 1, 2
order by 3 desc;
