import { Column, Result } from "../../lib/dashboard";
import { formatValue } from "../../lib/render";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../tremor/Select";

type DropdownProps = {
  headers: Column[];
  data: Result['queries'][0]['rows'];
  onChange: (newVars: Record<string, string | string[]>) => void;
  vars?: Record<string, string | string[]>;
};

function DashboardDropdown({ data, headers, onChange, vars }: DropdownProps) {
  if (!data) {
    return <div>No data</div>;
  }
  const valueIndex = headers.findIndex((header) => header.tag === "value");
  const labelIndex = headers.findIndex((header) => header.tag === "label");
  const varName = headers[valueIndex].name;
  const varField = (vars ?? {})[varName]
  const selectedValue = Array.isArray(varField) ? varField[0] : varField
  return (
    <>
      <Select
        defaultValue={formatValue(data[0][valueIndex])}
        onValueChange={(value) => {
          onChange({ ...vars, [varName]: value });
        }}
        value={selectedValue}
      >
        <SelectTrigger className="mx-auto">
          <SelectValue placeholder="Select" />
        </SelectTrigger>
        <SelectContent>
          {data.map((row) => (
            <SelectItem
              key={formatValue(row[valueIndex])}
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
