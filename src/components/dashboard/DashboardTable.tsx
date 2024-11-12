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
import { Card } from "../tremor/Card";
import { cx } from "../../lib/utils";

type TableProps = {
  label?: string;
  headers: Column[];
  data?: Result['sections'][0]['queries'][0]['rows']
  sectionCount: number;
};

function DashboardTable({ label, headers, data, sectionCount }: TableProps) {
  return (
    <div className={cx({
      "p-2 mb-3": true,
      "col-span-2": headers.length === 2 && sectionCount >= 2,
      "col-span-4": headers.length >= 3 || sectionCount === 1,
    })}>
      {label &&
        <h2 className="text-sm mb-2 text-center">
          {label}
        </h2>
      }
      <Card className="p-2">
        {!data ?
          (
            <div className={cx({
              "h-20 py-1 px-3 flex items-center justify-center text-slate-600": true,
              ["h-[calc(50vh-7.35rem)]"]: label,
              ["h-[calc(50vh-5.6rem)]"]: !label,
            })}>
              no data
            </div>
          ) :
          <TableRoot className={cx({
            "overflow-auto": true,
            ["max-h-[calc(100vh-9.4rem)]"]: label,
            ["max-h-[calc(100vh-7.6rem)]"]: !label,
          })}>
            <Table>
              <TableHead className="sticky top-0 bg-white shadow-sm">
                <TableRow>
                  {headers.map((header) => (
                    <TableHeaderCell key={header.name}>{header.name}</TableHeaderCell>
                  ))}
                </TableRow>
              </TableHead>
              <TableBody>
                {
                  data.map((items, index) => (
                    <TableRow key={index} className={index % 2 === 0 ? "bg-slate-50" : undefined}>
                      {items.map((item, index) => {
                        const classes =
                          typeof item === "number" ? "text-right" : "text-left";
                        return (
                          <TableCell key={index} className={classes}>
                            {formatValue(item, headers[index].type)}
                          </TableCell>
                        );
                      })}
                    </TableRow>
                  ))
                }
              </TableBody>
            </Table>
          </TableRoot>
        }
      </Card>
    </div>
  );
}

export default DashboardTable;
