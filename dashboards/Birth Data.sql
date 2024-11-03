-- create table base as (
  -- select
    -- Entity as country,
    -- Year as year,
    -- case when year > 2023 then "Deaths - Sex: all - Age: all - Variant: medium" else "Deaths - Sex: all - Age: all - Variant: estimates" end as deaths,
    -- case when year > 2023 then "Births - Sex: all - Age: all - Variant: medium" else "Births - Sex: all - Age: all - Variant: estimates" end as births
  -- from 'births-and-deaths-projected-to-2100.csv'
  -- where deaths is not null and births is not null
-- )



select 'Global Rate by Year';
select
  make_date(year, 1, 1) as Year,
  round(sum(births) / sum(deaths), 2) as Ratio
from base
group by year
order by year;

select '2024 by Country';
select
  country as Country,
  sum(births)::int64 as Births,
  sum(deaths)::int64 as Deaths,
  round(sum(births) / sum(deaths), 2) as Ratio
from base
where year = 2024
group by country
order by ratio;

