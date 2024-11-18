import { RiArrowRightUpLine, RiArrowRightDownLine } from "@remixicon/react";
import { Column, Result } from "../../lib/dashboard";

import { formatValue } from "../../lib/render";
import { Card } from "../tremor/Card";
import { cx } from "../../lib/utils";

type ValueProps = {
  label?: string;
  headers: Column[];
  data: Result['sections'][0]['queries'][0]['rows']
  sectionCount: number;
};

function DashboardValue({ label, headers, data, sectionCount }: ValueProps) {
  const valueIndex = headers.findIndex(header => header.tag === 'value')
  const valueHeader = headers[valueIndex]
  const value = data[0][valueIndex]
  const compareIndex = headers.findIndex(header => header.tag === 'compare')
  const compareHeader = compareIndex !== -1 ? headers[compareIndex] : undefined
  const compareValue = compareIndex !== -1 ? data[0][compareIndex] : undefined
  const percent = typeof value === 'number' && typeof compareValue === 'number' && compareValue !== value ?
    Math.round(-1 * (1 - (value / compareValue)) * 100) : undefined
  return (
    <div className="p-2 mb-3">
      {label ?
        <h2 className="text-md mb-2 text-center text-slate-700">
          {label}
        </h2>
        : <div className="h-8"></div>
      }
      <Card className={cx(
        "overflow-auto px-2 py-16 flex items-center justify-center", {
        "min-h-[250px]": label,
        "min-h-[calc(250px+2.00rem)]": !label,
        ["sm:h-[calc(36vh)] lg:h-[calc(60vh-4.1rem)] xl:h-[calc(100vh-8.25rem)]"]: label && sectionCount === 2,
        ["sm:h-[calc(36vh)] lg:h-[calc(60vh-2.1rem)] xl:h-[calc(100vh-6.25rem)]"]: !label && sectionCount === 2,
        ["md:h-[calc(39vh)] 2xl:h-[calc(50vh)]"]: label && sectionCount === 3,
        ["md:h-[calc(43vh)] 2xl:h-[calc(54vh)]"]: !label && sectionCount === 3,
        ["md:h-[calc(50vh-5.7rem)]"]: sectionCount >= 4,
      })}>
        <div className="text-center">
          <div className={"text-6xl text-slate-800"}>
            {formatValue(value, valueHeader.type)}
          </div>
          <div className="text-lg text-slate-800">
            {valueHeader.name}
          </div>
          {compareValue && compareHeader ? (
            <div className="text-sm text-slate-800 mt-2 flex items-center">
              <span>{compareHeader.name}:</span>
              <span className="ml-1">{formatValue(compareValue, compareHeader.type)}</span>
              {percent && <div
                className={cx(
                  "rounded px-1 py-1 ml-1 text-sm font-medium text-white flex flex-nowrap items-center",
                  {
                    "bg-emerald-500": percent >= 0,
                    "bg-red-500": percent < 0,
                  }
                )}
              >{percent > 0 && '+'}{percent}%{percent > 0 ? <RiArrowRightUpLine className="ml-1 size-4 shrink-0 text-white" /> : <RiArrowRightDownLine className="ml-1 size-4 shrink-0 text-white" />}</div>}
            </div>
          ) : undefined}
        </div>
      </Card>
    </div>
  );
}

export default DashboardValue;

