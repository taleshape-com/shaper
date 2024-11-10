import { Column, Result } from "../../lib/dashboard";
import { Button } from "../tremor/Button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "../tremor/DropdownMenu";
import { formatValue } from "../../lib/render";

type DropdownProps = {
  label?: string;
  headers: Column[];
  data: Result['queries'][0]['rows'];
  onChange: (newVars: Record<string, string | string[]>) => void;
  vars?: Record<string, string | string[]>;
};

function DashboardDropdownMulti({
  label,
  data,
  headers,
  onChange,
  vars,
}: DropdownProps) {
  if (!data) {
    return <div>No data</div>;
  }
  const valueIndex = headers.findIndex((header) => header.tag === "value");
  const labelIndex = headers.findIndex((header) => header.tag === "label");
  const hintIndex = headers.findIndex((header) => header.tag === "hint");
  const varName = headers[valueIndex].name;
  const selectedVal = (vars ?? {})[varName];
  const selectedValArr = selectedVal ? (Array.isArray(selectedVal)
    ? selectedVal
    : [selectedVal]) : [];
  return (
    <div className="flex justify-center">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="secondary">{label ?? "Pick"}</Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          {data.map((row) => {
            const value = formatValue(row[valueIndex]);
            return (
              <DropdownMenuCheckboxItem
                key={value}
                checked={!selectedVal || selectedValArr.includes(value)}
                onCheckedChange={(checked) => {
                  const valSet = new Set(selectedValArr);
                  if (!selectedVal || checked) {
                    valSet.add(value);
                  } else {
                    valSet.delete(value);
                  }
                  onChange({ ...vars, [varName]: Array.from(valSet) });
                }}
                hint={
                  hintIndex !== -1 ? formatValue(row[hintIndex]) : undefined
                }
              >
                {row[labelIndex !== -1 ? labelIndex : valueIndex]}
              </DropdownMenuCheckboxItem>
            );
          })}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

export default DashboardDropdownMulti;
