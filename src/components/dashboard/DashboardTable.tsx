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
  data: Result['sections'][0]['queries'][0]['rows']
  sectionCount: number;
};

function DashboardTable({ label, headers, data, sectionCount }: TableProps) {
  return (
    <div className={cx({
      "p-2 mb-3": true,
      "col-span-2": headers.length >= 3 && sectionCount >= 5,
      "col-span-4": headers.length >= 6 || (headers.length >= 5 && sectionCount <= 4) || sectionCount === 1,
    })}>
      {label &&
        <h2 className="text-md mb-2 text-center text-slate-700">
          {label}
        </h2>
      }
      <Card className="px-2 pt-2 pb-4">
        <TableRoot className={cx({
          "overflow-auto": true,
          ["max-h-[calc(100vh-9.4rem)] lg:min-h-[calc(60vh-4.1rem)] xl:min-h-[calc(100vh-9.8rem)]"]: label && sectionCount === 2,
          ["max-h-[calc(100vh-7.6rem)] lg:min-h-[calc(60vh-2.1rem)]"]: !label && sectionCount === 2,
          ["max-h-[calc(100vh-9.4rem)] md:min-h-[calc(50vh-1.6rem)]"]: label && sectionCount === 3,
          ["max-h-[calc(100vh-7.6rem)] md:min-h-[calc(50vh+0.45rem)]"]: !label && sectionCount === 3,
          ["max-h-[calc(100vh-9.4rem)] md:min-h-[calc(50vh-7.4rem)]"]: label && sectionCount >= 4,
          ["max-h-[calc(100vh-7.6rem)] md:min-h-[calc(50vh-5.3rem)]"]: !label && sectionCount >= 4,
        })}>
          <Table>
            <TableHead className="sticky top-0 bg-white shadow-sm">
              <TableRow>
                {headers.map((header) => (
                  <TableHeaderCell className={header.type === 'number' ? 'text-center' : 'text-left'} key={header.name}>{header.name}</TableHeaderCell>
                ))}
              </TableRow>
            </TableHead>
            <TableBody>
              {
                data.map((items, index) => (
                  <TableRow key={index} className={index % 2 === 0 ? "bg-slate-50" : undefined}>
                    {items.map((item, index) => {
                      const headerType = headers[index].type
                      const classes = headerType === 'number' ? 'text-center' : 'text-left';
                      return (
                        <TableCell key={index} className={classes}>
                          {formatValue(item, headerType)}
                        </TableCell>
                      );
                    })}
                  </TableRow>
                ))
              }
            </TableBody>
          </Table>
        </TableRoot>
      </Card>
    </div>
  );
}

export default DashboardTable;
