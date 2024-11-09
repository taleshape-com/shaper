import { Column } from "../../lib/dashboard";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../tremor/Select";

type DropdownProps = {
  headers: Column[];
  data: (string | number)[][];
  onChange: (value: string, varName: string) => void;
  vars?: Record<string, string>;
};

const formatValue = (value: string | number): string => {
  if (typeof value === "number") {
    return value.toLocaleString();
  }
  return value;
};

function DashboardDropdown({ data, headers, onChange, vars }: DropdownProps) {
  if (!data) {
    return <div>No data</div>;
  }
  const valueIndex = headers.findIndex((header) => header.tag === "value");
  const labelIndex = headers.findIndex((header) => header.tag === "label");
  const varName = headers[valueIndex].name;
  return (
    <>
      <Select
        defaultValue={formatValue(data[0][valueIndex])}
        onValueChange={(value) => onChange(value, varName)}
        value={vars && vars[varName]}
      >
        <SelectTrigger className="mx-auto">
          <SelectValue placeholder="Select" />
        </SelectTrigger>
        <SelectContent>
          {data.map((row) => (
            <SelectItem
              key={row[valueIndex]}
              value={formatValue(row[valueIndex])}
            >
              {row[labelIndex !== -1 ? labelIndex : valueIndex]}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </>
  );
}

export default DashboardDropdown;
