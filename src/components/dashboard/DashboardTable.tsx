import { RiArrowRightUpLine, RiArrowRightDownLine } from "@remixicon/react";
import { Column, Result } from "../../lib/dashboard";
import { formatValue } from "../../lib/render";
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
        <TableHead className="sticky top-0 bg-white shadow-sm">
          <TableRow>
            {headers.map((header) => (
              <TableHeaderCell className="text-center" key={header.name}>{header.name}</TableHeaderCell>
            ))}
          </TableRow>
        </TableHead>
        <TableBody>
          {
            data.map((items, index) => (
              <TableRow key={index} className={index % 2 === 0 ? "bg-slate-50" : undefined}>
                {items.map((item, index) => {
                  const header = headers[index]
                  const percent = header.tag === 'trend' && typeof item === 'number' ? Math.round(-100 * (1 - item)) : undefined
                  return (
                    <TableCell key={index} className="text-center">
                      {percent !== undefined ? percent === 0 ? '-' : (
                        <div
                          className={cx(
                            "ml-2 rounded px-1 py-1 text-sm font-medium text-white flex flex-nowrap items-center justify-center bg-slate-800",
                            // { "bg-emerald-500": percent >= 0, "bg-red-500": percent < 0, }
                          )}
                        >
                          {percent > 0 && '+'}{percent}%{
                            percent > 0 ?
                              <RiArrowRightUpLine className="ml-1 size-4 shrink-0 text-white" />
                              : <RiArrowRightDownLine className="ml-1 size-4 shrink-0 text-white" />
                          }
                        </div>) :
                        formatValue(item, header.type)
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

export default DashboardTable;
