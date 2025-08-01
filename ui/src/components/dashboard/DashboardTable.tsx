// SPDX-License-Identifier: MPL-2.0

import { RiArrowRightUpLine, RiArrowRightDownLine } from "@remixicon/react";
import { Column, Result } from "../../lib/dashboard";
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

type TableProps = {
  headers: Column[];
  data: Result['sections'][0]['queries'][0]['rows']
};

function DashboardTable({ headers, data }: TableProps) {
  return (
    <TableRoot className="h-full">
      <Table>
        <TableHead className="sticky top-0 bg-cbgs dark:bg-dbgs shadow-sm">
          <TableRow>
            {headers.map((header) => (
              <TableHeaderCell
                className={cx("text-ctext dark:text-dtext", {
                  "text-right": alignRight(header),
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
                  const header = headers[index]
                  const percent = header.tag === 'trend' && typeof item === 'number' ? Math.round(-100 * (1 - item)) : undefined
                  const formattedValue = percent !== undefined ? '' : formatValue(item, header.type, true).toString()
                  return (
                    <TableCell key={index} className={cx("text-ctext dark:text-dtext", { "text-right": alignRight(header) })}>
                      {percent !== undefined ? percent === 0 ? '-' : (
                        <div
                          className={cx(
                            "ml-2 rounded px-1 py-1 text-sm font-medium flex flex-nowrap items-center justify-center text-ctexti bg-cbgi dark:text-dtexti dark:bg-dbgi",
                            // { "bg-emerald-500": percent >= 0, "bg-red-500": percent < 0, }
                          )}
                        >
                          {percent > 0 && '+'}{percent}%{
                            percent > 0 ?
                              <RiArrowRightUpLine className="ml-1 size-4 shrink-0 text-ctexti dark:text-dtexti" />
                              : <RiArrowRightDownLine className="ml-1 size-4 shrink-0 text-ctexti dark:text-dtexti" />
                          }
                        </div>) :
                        <span className={cx({
                          "font-display": !isJSONType(header.type),
                          "font-mono": isJSONType(header.type) || header.type === 'number' || header.type === 'boolean' || header.type === 'percent',
                          "text-xs": formattedValue.length > 30,
                        })}>
                          {formattedValue}
                        </span>
                      }
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

const alignRight = (header: Column) => {
  return header.type === "number" || header.type === "percent" || header.type === "duration" || header.type === "boolean";
};

export default DashboardTable;
