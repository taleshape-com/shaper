import { RiArrowRightUpLine, RiArrowRightDownLine } from "@remixicon/react";
import { Column, Result } from "../../lib/dashboard";

import { formatValue, isJSONType } from "../../lib/render";
import { cx, getNameIfSet } from "../../lib/utils";

type ValueProps = {
  headers: Column[];
  data: Result['sections'][0]['queries'][0]['rows']
};

function DashboardValue({ headers, data }: ValueProps) {
  const valueIndex = headers.findIndex(header => header.tag === 'value')
  const valueHeader = headers[valueIndex]
  const value = data[0][valueIndex]
  const compareIndex = headers.findIndex(header => header.tag === 'compare')
  // TODO: Currently we format compare value by the value header type, but we should use the compare header type once ::COMPARE supports multiple data types
  const compareHeader = compareIndex !== -1 ? headers[compareIndex] : undefined
  const compareValue = compareIndex !== -1 ? data[0][compareIndex] : undefined
  const percent = typeof value === 'number' && typeof compareValue === 'number' && compareValue !== value ?
    Math.round(-100 * (1 - (value / compareValue))) : undefined
  const formattedValue = formatValue(value, valueHeader.type, true).toString()
  const hasLabel = valueHeader.name !== value && valueHeader.name !== `'${value}'`

  return (
    <div className="items-center h-full flex flex-col justify-center text-center overflow-auto">
      <div className={cx("font-semibold", {
        "font-mono": isJSONType(valueHeader.type),
        "text-xs": formattedValue.length >= 47,
        "text-sm": formattedValue.length < 47 && formattedValue.length >= 42,
        "text-lg": formattedValue.length < 38 && formattedValue.length >= 34,
        "text-xl": formattedValue.length < 34 && formattedValue.length >= 29,
        "text-2xl": formattedValue.length < 29 && formattedValue.length >= 25,
        "text-3xl": formattedValue.length < 25 && formattedValue.length >= 21,
        "text-4xl": formattedValue.length < 21 && formattedValue.length >= 17,
        "text-5xl": formattedValue.length < 17 && formattedValue.length >= 13,
        "text-6xl": formattedValue.length < 13 && formattedValue.length >= 6,
        "text-7xl": formattedValue.length < 6,
      })}>
        {formattedValue}
      </div>
      {hasLabel && getNameIfSet(valueHeader.name) && (
        <div className={cx("mt-3 font-medium font-display", {
          "text-xs": valueHeader.name.length >= 40,
          "text-sm": valueHeader.name.length < 40 && valueHeader.name.length >= 35,
          "text-lg": valueHeader.name.length < 30 && valueHeader.name.length >= 20,
          "text-xl": valueHeader.name.length < 20,
        })}>
          {valueHeader.name}
        </div>
      )}
      {compareValue && compareHeader ? (
        <div className="text-sm mt-2 flex items-center justify-center font-medium">
          <span>{compareHeader.name}:</span>
          <span className="ml-1">{formatValue(compareValue, valueHeader.type, true)}</span>
          {percent && <div
            className={cx(
              "ml-2 rounded px-1 py-1 text-sm font-medium text-ctexti dark:text-dtexti flex flex-nowrap items-center b bg-cbgi dark:bg-dbgi",
              // { "bg-emerald-500": percent >= 0, "bg-red-500": percent < 0, }
            )}
          >{percent > 0 && '+'}{percent}%{percent > 0 ? <RiArrowRightUpLine className="ml-1 size-4 shrink-0 text-ctexti dark:text-dtexti" /> : <RiArrowRightDownLine className="ml-1 size-4 shrink-0 text-ctexti dark:text-dtexti" />}</div>}
        </div>
      ) : undefined}
    </div>
  );
}

export default DashboardValue;

