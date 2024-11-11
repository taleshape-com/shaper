import { Column, Result } from "../../lib/dashboard";
import { formatValue } from "../../lib/render";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../tremor/Select";
import { Label } from "../tremor/Label"
import { cx } from "../../lib/utils";

type DropdownProps = {
  label?: string;
  headers: Column[];
  data?: Result['queries'][0]['rows'];
  onChange: (newVars: Record<string, string | string[]>) => void;
  vars?: Record<string, string | string[]>;
};

function DashboardDropdown({ label, data, headers, onChange, vars }: DropdownProps) {
  if (!data) {
    return null;
  }
  const valueIndex = headers.findIndex((header) => header.tag === "value");
  const labelIndex = headers.findIndex((header) => header.tag === "label");
  const varName = headers[valueIndex].name;
  const varField = (vars ?? {})[varName]
  const selectedValue = Array.isArray(varField) ? varField[0] : varField
  return (
    <>
      {label && <Label htmlFor={label} className="ml-2 pr-2">{label}:</Label>}
      <div className={cx({ ["ml-2"]: !label })}>
        <Select
          defaultValue={formatValue(data[0][valueIndex]).toString()}
          onValueChange={(value) => {
            onChange({ ...vars, [varName]: value });
          }}
          value={selectedValue}
        >
          <SelectTrigger
            id={label}
            className="mx-auto"
          >
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {data.map((row) => (
              <SelectItem
                key={formatValue(row[valueIndex])}
                value={formatValue(row[valueIndex]).toString()}
              >
                {row[labelIndex !== -1 ? labelIndex : valueIndex]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </>
  );
}

export default DashboardDropdown;
