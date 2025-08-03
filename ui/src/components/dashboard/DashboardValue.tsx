// SPDX-License-Identifier: MPL-2.0

import React, { useRef, useState, useEffect } from "react";
import { RiArrowRightUpLine, RiArrowRightDownLine } from "@remixicon/react";
import { Column, Result } from "../../lib/dashboard";

import { formatValue, isJSONType } from "../../lib/render";
import { cx, getNameIfSet } from "../../lib/utils";
import TextWithLinks from "../TextWithLinks";

type ValueProps = {
  headers: Column[];
  data: Result['sections'][0]['queries'][0]['rows'];
  yScroll: boolean;
};

const getLongestLineLength = (text: string) => {
  return Math.max(...text.split('\n').map(line => line.length));
};

const calcFontSize = (width: number, longestLine: number, factor: number, min: number, max: number, round: number) => {
  if (!width || !longestLine) return min;
  return Math.max(min, Math.min(max, Math.floor((width / longestLine) * factor / round) * round));
};

function DashboardValue({ headers, data, yScroll }: ValueProps) {
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

  const containerRef = useRef<HTMLDivElement>(null);
  const [containerWidth, setContainerWidth] = useState(0);

  useEffect(() => {
    const node = containerRef.current;
    if (!node) return;
    const updateWidth = () => setContainerWidth(node.offsetWidth);
    updateWidth();
    const ro = new window.ResizeObserver(updateWidth);
    ro.observe(node);
    return () => ro.disconnect();
  }, []);

  const valueLongestLine = getLongestLineLength(formattedValue);
  const valueFontSize = calcFontSize(containerWidth, valueLongestLine, 1.6, 16, 64, 8);

  const labelText = hasLabel && getNameIfSet(valueHeader.name) ? valueHeader.name : '';
  const labelLongestLine = getLongestLineLength(labelText);
  const labelFontSize = calcFontSize(containerWidth, labelLongestLine, 1.2, 16, 24, 4);

  return (
    <div
      className={cx(
        "items-center h-full w-full flex flex-col justify-center overflow-x-auto overflow-y-hidden",
        { "overflow-y-auto": yScroll },
      )}
      ref={containerRef}
    >
      <div
        className={cx({
          "font-mono": isJSONType(valueHeader.type),
          "font-semibold": formattedValue.length < 300,
          "text-center": formattedValue.length < 400,
          "text-justify": formattedValue.length >= 400,
        })}
        style={{ fontSize: `${valueFontSize}px`, lineHeight: 1.2 }}
      >
        {typeof formattedValue === 'string' && formattedValue.includes('\n')
          ? formattedValue.split('\n').map((line, idx, arr) => (
            <React.Fragment key={idx}>
              <TextWithLinks text={line} />
              {idx < arr.length - 1 && <br />}
            </React.Fragment>
          ))
          : <TextWithLinks text={formattedValue} />}
      </div>
      {
        hasLabel && getNameIfSet(valueHeader.name) && (
          <div
            className={cx("mt-3 font-medium font-display text-center")}
            style={{ fontSize: `${labelFontSize}px`, lineHeight: 1.2 }}
          >
            {valueHeader.name}
          </div>
        )
      }
      {
        compareValue && compareHeader ? (
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
        ) : undefined
      }
    </div >
  );
}

export default DashboardValue;

