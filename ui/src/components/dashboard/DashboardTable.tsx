// SPDX-License-Identifier: MPL-2.0

import { useMemo } from "react";
import { RiArrowRightUpLine, RiArrowRightDownLine } from "@remixicon/react";
import { Column } from "../../lib/types";
import { formatValue, isJSONType } from "../../lib/render";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRoot,
  TableRow,
} from "../tremor/Table";
import { cx } from "../../lib/utils";
import TextWithLinks from "../TextWithLinks";

type TableProps = {
  headers: Column[];
  data: (string | number | boolean)[][];
};

function DashboardTable ({ headers, data }: TableProps) {
  const isSmall = useMemo(() => {
    return data.some((row) =>
      row?.some((item, index) => isCellLong(item, headers[index])),
    );
  }, [data, headers]);

  if (data.length === 1) {
    const row = data[0];
    return (
      <TableRoot className="h-full overflow-auto">
        <Table className={cx(isSmall && "text-xs")}>
          <TableBody>
            {headers.map((header, index) => {
              const item = row?.[index];
              return (
                <TableRow key={header.name}>
                  <TableHeaderCell
                    scope="row"
                    className={cx(
                      "text-ctext dark:text-dtext py-2.5 font-medium font-display whitespace-nowrap w-px pr-6 font-semibold",
                      { "text-xs": isSmall },
                    )}
                  >
                    {header.name}
                  </TableHeaderCell>
                  <TableCell
                    className={cx(
                      "text-ctext dark:text-dtext py-2.5 whitespace-normal",
                      { "text-xs": isSmall },
                    )}
                  >
                    {renderCellContent(header, item, true, isSmall)}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </TableRoot>
    );
  }

  return (
    <TableRoot className="h-full overflow-auto">
      <Table className={cx(isSmall && "text-xs")}>
        <TableHead className="z-10">
          <TableRow>
            {headers.map((header) => (
              <TableHeaderCell
                className={cx("text-ctext dark:text-dtext z-10 sticky top-0", {
                  "text-right": alignRight(header),
                  "text-xs": isSmall,
                })}
                key={header.name}
              >{header.name}</TableHeaderCell>
            ))}
          </TableRow>
        </TableHead>
        <TableBody>
          {
            data.map((items, index) => (
              <TableRow key={index}>
                {items.map((item, index) => {
                  const header = headers[index];
                  return (
                    <TableCell
                      key={index}
                      className={cx("text-ctext dark:text-dtext", {
                        "text-right": alignRight(header),
                        "text-xs": isSmall,
                      })}
                    >
                      {renderCellContent(header, item, false, isSmall)}
                    </TableCell>
                  );
                })}
              </TableRow>
            ))
          }
        </TableBody>
      </Table>
    </TableRoot>
  );
}

const isCellLong = (
  item: string | number | boolean | null | undefined,
  header?: Column,
) => {
  if (!header) return false;
  if (header.tag === "trend" && typeof item === "number") {
    return false;
  }
  const formattedValue = formatValue(item, header.type, true).toString();
  return formattedValue.length > 30;
};

const renderCellContent = (
  header: Column,
  item: string | number | boolean,
  isPivoted?: boolean,
  isSmall?: boolean,
) => {
  let percent = undefined;
  let percentDisplay = undefined;

  if (header.tag === "trend" && typeof item === "number") {
    // Calculate the percentage change: -100 * (1 - item) = 100 * (item - 1)
    const percentValue = -100 * (1 - item);

    // Format based on the specified ranges
    if (percentValue > -0.0001 && percentValue < 0.0001) {
      percent = percentValue; // Keep the actual value to determine direction
      percentDisplay = "0";
    } else if (percentValue > -1 && percentValue < 1) {
      percent = Math.round(percentValue * 10000) / 10000; // Round to 4 decimal places
      percentDisplay = percent.toString();
    } else if (percentValue > -10 && percentValue < 10) {
      percent = Math.round(percentValue * 100) / 100; // Round to 2 decimal places
      percentDisplay = percent.toString();
    } else {
      percent = Math.round(percentValue); // Round without decimal places
      percentDisplay = percent.toString();
    }
  }

  const formattedValue =
    percent !== undefined
      ? ""
      : formatValue(item, header.type, true).toString();

  if (percent !== undefined) {
    if (percent === 0) return "-";
    return (
      <div
        className={cx(
          "rounded px-1.5 py-1 font-medium flex flex-nowrap items-center justify-center text-ctexti bg-cbgi dark:text-dtexti dark:bg-dbgi opacity-55",
          isSmall ? "text-xs" : "text-sm",
          isPivoted ? "inline-flex w-fit -my-0.5" : "ml-2 -my-1 max-w-28 ml-auto",
        )}
      >
        {percent > 0 ? "+" : ""}
        {percentDisplay}%{
          percent > 0 ? (
            <RiArrowRightUpLine className="ml-1 size-4 shrink-0 text-ctexti dark:text-dtexti" />
          ) : (
            <RiArrowRightDownLine className="ml-1 size-4 shrink-0 text-ctexti dark:text-dtexti" />
          )
        }
      </div>
    );
  }

  return (
    <span
      className={cx({
        "font-display": !isJSONType(header.type),
        "font-mono":
          isJSONType(header.type) ||
          header.type === "number" ||
          header.type === "boolean" ||
          header.type === "percent",
        "text-xs": isSmall,
      })}
    >
      <TextWithLinks text={formattedValue} />
    </span>
  );
};

const alignRight = (header: Column) => {
  return header.type === "number" || header.type === "percent" || header.type === "duration" || header.type === "boolean";
};

export default DashboardTable;
