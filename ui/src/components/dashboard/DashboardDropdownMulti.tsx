// SPDX-License-Identifier: MPL-2.0

import * as SelectPrimitives from "@radix-ui/react-select";
import { RiExpandUpDownLine } from "@remixicon/react";
import { Column, Result } from "../../lib/types";
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
  data: Result['sections'][0]['queries'][0]['rows']
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
  const valueIndex = headers.findIndex((header) => header.tag === "value");
  const labelIndex = headers.findIndex((header) => header.tag === "label");
  const hintIndex = headers.findIndex((header) => header.tag === "hint");
  const varName = headers[valueIndex].name;
  const selectedVal = (vars ?? {})[varName];
  const selectedValArr = selectedVal ? (Array.isArray(selectedVal)
    ? selectedVal
    : [selectedVal]) : [];
  return (
    <div className="ml-2 select-none">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="secondary" className="flex w-full items-center justify-between my-1 data-[state=open]:bg-cbga data-[state=open]:dark:bg-dbga">
            {label ?? varName} ({selectedVal ? selectedValArr.length : data.length})
            <SelectPrimitives.Icon asChild>
              <RiExpandUpDownLine
                className="ml-2 size-4 shrink-0 text-ctext2 dark:text-dtext2"
              />
            </SelectPrimitives.Icon>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          {data.map((row) => {
            const value = formatValue(row[valueIndex], headers[valueIndex].type);
            return (
              <DropdownMenuCheckboxItem
                key={value}
                checked={!selectedVal || selectedValArr.includes(value.toString())}
                onCheckedChange={(checked) => {
                  const valSet = new Set(selectedValArr);
                  if (!selectedVal || checked) {
                    valSet.add(value.toString());
                  } else {
                    valSet.delete(value.toString());
                  }
                  onChange({ ...vars, [varName]: Array.from(valSet) });
                }}
                hint={
                  hintIndex !== -1 ? formatValue(row[hintIndex], headers[hintIndex].type, true).toString() : undefined
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
