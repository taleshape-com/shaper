select
  make_date(year, 1, 1) as Year,
  round(sum(births) / sum(deaths), 2) as Ratio
from base
group by year
order by year;

select
  country as Country,
  sum(births)::int64 as Births,
  sum(deaths)::int64 as Deaths,
  round(sum(births) / sum(deaths), 2) as Ratio
from base
where year = 2024
group by country
order by ratio;

