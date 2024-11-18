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
                  const headerType = headers[index].type
                  return (
                    <TableCell key={index} className="text-center">
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
  );
}

export default DashboardTable;
