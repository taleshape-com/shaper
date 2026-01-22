// SPDX-License-Identifier: MPL-2.0

import React, { useRef, useState, useEffect } from "react";
import { RiArrowRightUpLine, RiArrowRightDownLine } from "@remixicon/react";
import { Column } from "../../lib/types";

import { formatValue, isJSONType } from "../../lib/render";
import { isDatableType } from "../../lib/types";
import { cx, getNameIfSet } from "../../lib/utils";
import TextWithLinks from "../TextWithLinks";

type ValueProps = {
  headers: Column[];
  data: (string | number | boolean)[][];
};

const calcFontSize = (width: number, text: string, max: number) => {
  const min = 16;
  if (!width || !text.length || text.includes("\n")) return min;
  const res = Math.max(min, Math.min(max, (width / text.length) * 1.8));
  if (res <= 24) {
    return min;
  }
  if (res < 32) {
    return 24;
  }
  return max;
};

const getComparePercent = (
  value: number | string | boolean | undefined,
  compareValue: number | string | boolean | undefined,
) => {
  if (typeof value !== "number" || typeof compareValue !== "number" || compareValue === value || compareValue === 0) {
    return undefined;
  }
  const percent = -100 * (1 - (value / compareValue));
  if (percent > -0.0001 && percent < 0.0001) {
    return undefined;
  }
  if (percent > -0.01 && percent < 0.01) {
    return Math.round(percent * 10000) / 10000;
  }
  if (percent > -10 && percent < 10) {
    return Math.round(percent * 100) / 100;
  }
  return Math.round(percent);
};

function DashboardValue ({ headers, data }: ValueProps) {
  const valueIndex = headers.findIndex(header => header.tag === "value");
  const valueHeader = headers[valueIndex];
  const value = data[0][valueIndex];
  const compareIndex = headers.findIndex(header => header.tag === "compare");
  // TODO: Currently we format compare value by the value header type, but we should use the compare header type once ::COMPARE supports multiple data types
  const compareHeader = compareIndex !== -1 ? headers[compareIndex] : undefined;
  const compareValue = compareIndex !== -1 ? data[0][compareIndex] : undefined;
  const percent = getComparePercent(value, compareValue);
  const formattedValue = formatValue(value, valueHeader.type, true).toString();
  const label = getNameIfSet(valueHeader.name);
  const hasLabel = !!label && label !== value && label !== formattedValue && label !== `'${value}'`;

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

  const maxFontSize = valueHeader.type === "number" || valueHeader.type === "percent" || valueHeader.type === "boolean"
    ? 40
    : isDatableType(valueHeader.type)
      ? 36
      : 32;
  const valueFontSize = calcFontSize(containerWidth, formattedValue, maxFontSize);

  const labelFontSize = valueFontSize === 16 ? 14 : 18;

  return (
    <div
      className={cx("h-full w-full flex flex-col", { "justify-center": hasLabel || valueFontSize > 16 })}
      ref={containerRef}
    >
      <div
        className={cx("overflow-auto py-2", {
          "font-mono": isJSONType(valueHeader.type),
          "font-semibold text-center": hasLabel || (formattedValue.length <= 100 && !formattedValue.includes("\n")),
          "text-justify": formattedValue.length >= 300,
        })}
        style={{ fontSize: `${valueFontSize}px`, lineHeight: 1.2 }}
      >
        {typeof formattedValue === "string" && formattedValue.includes("\n")
          ? formattedValue.split("\n").map((line, idx, arr) => (
            <React.Fragment key={idx}>
              <TextWithLinks text={line} />
              {idx < arr.length - 1 && <br />}
            </React.Fragment>
          ))
          : <TextWithLinks text={formattedValue} />}
      </div>
      {
        hasLabel && (
          <div
            className={cx("font-medium font-display text-center")}
            style={{ fontSize: `${labelFontSize}px`, lineHeight: 1.2 }}
          >
            <TextWithLinks text={label} />
          </div>
        )
      }
      {
        compareValue !== undefined && compareHeader ? (
          <div className="text-sm mt-6 flex items-center justify-center font-medium">
            {getNameIfSet(compareHeader.name) && (
              <span>{getNameIfSet(compareHeader.name)}:</span>
            )}
            <span className="ml-1">{formatValue(compareValue, valueHeader.type, true)}</span>
            {percent !== undefined && <div
              className={cx(
                "ml-2 rounded px-1 py-1 text-sm font-medium text-ctexti dark:text-dtexti",
                "flex flex-nowrap items-center b bg-cbgi dark:bg-dbgi opacity-55",
              )}
            >{percent > 0 && "+"}{percent}%{percent > 0 ? <RiArrowRightUpLine className="ml-1 size-4 shrink-0 text-ctexti dark:text-dtexti" /> : <RiArrowRightDownLine className="ml-1 size-4 shrink-0 text-ctexti dark:text-dtexti" />}</div>}
          </div>
        ) : undefined
      }
    </div>
  );
}

export default DashboardValue;
