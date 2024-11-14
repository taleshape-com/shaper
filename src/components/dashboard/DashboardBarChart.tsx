import { Column, isTimeType, Result } from "../../lib/dashboard";
import { formatValue } from "../../lib/render";
import { BarChart } from "../tremor/BarChart";
import { Card } from "../tremor/Card";
import { cx } from "../../lib/utils";

type BarProps = {
  label?: string;
  headers: Column[];
  data: Result['sections'][0]['queries'][0]['rows']
  sectionCount: number;
  stacked?: boolean;
  vertical?: boolean;
};

const DashboardBarChart = ({ label, headers, data, sectionCount, stacked, vertical }: BarProps) => {
  if (data.length === 0) {
    return null;
  }
  const valueAxisHeader = headers.find((c) => c.tag === "value");
  if (!valueAxisHeader) {
    throw new Error("No column with tag 'value'");
  }
  const valueAxisName = valueAxisHeader.name;
  const categoryIndex = headers.findIndex((c) => c.tag === "category");
  const categories = new Set<string>();
  if (categoryIndex === -1) {
    categories.add(valueAxisName);
  }
  const indexAxisIndex = headers.findIndex((c) => c.tag === "index");
  const indexAxisHeader = headers[indexAxisIndex];
  const dataByIndexAxis = data.reduce(
    (acc, row) => {
      const key = formatValue(row[indexAxisIndex], indexAxisHeader.type);
      if (!acc[key]) {
        acc[key] = {};
      }
      row.forEach((cell, i) => {
        if (i === indexAxisIndex || i === categoryIndex) {
          return;
        }
        const c = formatValue(cell, headers[i].type)
        if (categoryIndex === -1) {
          acc[key][valueAxisName] = c;
          return;
        }
        const category = (row[categoryIndex] ?? '').toString();
        categories.add(category);
        acc[key][category] = c;
      });
      return acc;
    },
    {} as Record<string, Record<string, string | number>>,
  );
  const chartdata = Object.entries(dataByIndexAxis).map(([key, value]) => {
    return {
      [indexAxisHeader.name]: key,
      ...value,
    };
  });
  return (
    <div className="p-2 mb-2">
      {label ? <h2 className="text-sm mb-2 text-center">
        {label}
      </h2>
        : null
      }
      <Card className={cx({
        "py-1 px-3": true,
        "pt-10": categoryIndex === -1,
        "h-[calc(45vh)] sm:h-[calc(100vh-8.25rem)]": sectionCount === 1 && label,
        "h-[calc(45vh)] sm:h-[calc(100vh-6.5rem)]": sectionCount === 1 && !label,
        "h-[calc(50vh-6rem)] sm:h-[calc(100vh-10rem)] lg:h-[calc(55vh)] xl:h-[calc(100vh-8.25rem)]": sectionCount === 2 && label,
        "h-[calc(50vh-5rem)] sm:h-[calc(100vh-6.5rem)] lg:h-[calc(55vh+1.75rem)] xl:h-[calc(100vh-6.5rem)]": sectionCount === 2 && !label,
        "h-[calc(50vh-6.4rem)] sm:h-[calc(96vh-6.25rem)] md:h-[calc(50vh-5.75rem)]": sectionCount >= 3 && label,
        "h-[calc(50vh-4.6rem)] sm:h-[calc(96vh-4.5rem)] md:h-[calc(50vh-4rem)]": sectionCount >= 3 && !label,
        "2xl:h-[calc(50vh)]": sectionCount === 3 && label,
        "2xl:h-[calc(50vh+1.75rem)]": sectionCount === 3 && !label,
      })}>
        {!data ?
          (
            <div className="h-full py-1 px-3 flex items-center justify-center text-slate-600">
              no data
            </div>
          ) :
          <BarChart
            className="h-full"
            enableLegendSlider
            startEndOnly={chartdata.length > (vertical ? 20 : isTimeType(indexAxisHeader.type) ? 10 : 15)}
            type={stacked ? "stacked" : "default"}
            layout={vertical ? "vertical" : "horizontal"}
            data={chartdata}
            index={indexAxisHeader.name}
            categories={Array.from(categories)}
            valueFormatter={(number: number) => {
              return number.toLocaleString();
            }}
            xAxisLabel={vertical ? valueAxisName : isTimeType(indexAxisHeader.type) ? undefined : indexAxisHeader.name}
            yAxisLabel={vertical ? isTimeType(indexAxisHeader.type) ? undefined : indexAxisHeader.name : valueAxisName}
            showLegend={categoryIndex !== -1}
          />
        }
      </Card>
    </div>
  );
};

export default DashboardBarChart;
