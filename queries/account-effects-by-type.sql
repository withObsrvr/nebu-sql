select
  type,
  count(*) as n
from nebu('account-effects', start = 60200000, stop = 60200010)
group by 1
order by 2 desc;
