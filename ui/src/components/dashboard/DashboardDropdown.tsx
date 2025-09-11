// SPDX-License-Identifier: MPL-2.0

import { Column, Result } from "../../lib/types";
import { formatValue } from "../../lib/render";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../tremor/Select";
import { Label } from "../tremor/Label";
import { cx } from "../../lib/utils";

const EMPTY = '<<EMPTY>>';

type DropdownProps = {
  label?: string;
  headers: Column[];
  data: Result["sections"][0]["queries"][0]["rows"];
  onChange: (newVars: Record<string, string | string[]>) => void;
  vars?: Record<string, string | string[]>;
};

function DashboardDropdown({
  label,
  data,
  headers,
  onChange,
  vars,
}: DropdownProps) {
  const valueIndex = headers.findIndex((header) => header.tag === "value");
  const labelIndex = headers.findIndex((header) => header.tag === "label");
  const varName = headers[valueIndex].name;
  const varField = (vars ?? {})[varName];
  const selectedValue = Array.isArray(varField) ? varField[0] : varField;

  return (
    <>
      {label && (
        <Label htmlFor={label} className="ml-3 pr-1">
          {label}:
        </Label>
      )}
      <div className={cx("select-none", { ["ml-2"]: !label })}>
        <Select
          onValueChange={(value) => {
            if (value === EMPTY) {
              value = "";
            }
            onChange({ ...vars, [varName]: value });
          }}
          value={data.some((row) => row[valueIndex] === selectedValue) ? selectedValue : formatValue(
            data[0][valueIndex],
            headers[valueIndex].type,
          ).toString() || EMPTY}
        >
          <SelectTrigger
            id={label}
            className="mx-auto my-1 data-[state=open]:bg-cbga data-[state=open]:dark:bg-dbga"
          >
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {data.map((row) => (
              <SelectItem
                key={formatValue(row[valueIndex], headers[valueIndex].type)}
                value={formatValue(
                  row[valueIndex],
                  headers[valueIndex].type,
                ).toString() || EMPTY}
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
