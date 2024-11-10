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
  data: Result['queries'][0]['rows']
};

function DashboardTable({ headers, data }: TableProps) {
  if (!data) {
    return <div>No data</div>;
  }
  return (
    <TableRoot className="max-h-screen overflow-auto w-full">
      <Table>
        <TableHead>
          <TableRow>
            {headers.map((header) => (
              <TableHeaderCell key={header.name}>{header.name}</TableHeaderCell>
            ))}
          </TableRow>
        </TableHead>
        <TableBody>
          {data.map((items, index) => (
            <TableRow key={index}>
              {items.map((item, index) => {
                const classes =
                  typeof item === "number" ? "text-right" : "text-left";
                return (
                  <TableCell key={index} className={classes}>
                    {formatValue(item)}
                  </TableCell>
                );
              })}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableRoot>
  );
}

export default DashboardTable;
