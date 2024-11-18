import { RiArrowRightUpLine, RiArrowRightDownLine } from "@remixicon/react";
import { Column, Result } from "../../lib/dashboard";

import { formatValue } from "../../lib/render";
import { cx } from "../../lib/utils";

type ValueProps = {
  headers: Column[];
  data: Result['sections'][0]['queries'][0]['rows']
};

function DashboardValue({ headers, data }: ValueProps) {
  const valueIndex = headers.findIndex(header => header.tag === 'value')
  const valueHeader = headers[valueIndex]
  const value = data[0][valueIndex]
  const compareIndex = headers.findIndex(header => header.tag === 'compare')
  const compareHeader = compareIndex !== -1 ? headers[compareIndex] : undefined
  const compareValue = compareIndex !== -1 ? data[0][compareIndex] : undefined
  const percent = typeof value === 'number' && typeof compareValue === 'number' && compareValue !== value ?
    Math.round(-1 * (1 - (value / compareValue)) * 100) : undefined
  return (
    <div className="items-center h-full flex flex-col justify-center">
      <div className={"text-7xl text-slate-800"}>
        {formatValue(value, valueHeader.type)}
      </div>
      <div className="text-xl mt-1 text-slate-800">
        {valueHeader.name}
      </div>
      {compareValue && compareHeader ? (
        <div className="text-sm text-slate-800 mt-2 flex items-center justify-center">
          <span>{compareHeader.name}:</span>
          <span className="ml-1">{formatValue(compareValue, compareHeader.type)}</span>
          {percent && <div
            className={cx(
              "ml-2 rounded px-1 py-1 text-sm font-medium text-white flex flex-nowrap items-center bg-slate-800",
              // { "bg-emerald-500": percent >= 0, "bg-red-500": percent < 0, }
            )}
          >{percent > 0 && '+'}{percent}%{percent > 0 ? <RiArrowRightUpLine className="ml-1 size-4 shrink-0 text-white" /> : <RiArrowRightDownLine className="ml-1 size-4 shrink-0 text-white" />}</div>}
        </div>
      ) : undefined}
    </div>
  );
}

export default DashboardValue;

