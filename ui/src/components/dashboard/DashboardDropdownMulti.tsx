// SPDX-License-Identifier: MPL-2.0

import { useState } from "react";
import * as SelectPrimitives from "@radix-ui/react-select";
import { RiExpandUpDownLine } from "@remixicon/react";
import { Column } from "../../lib/types";
import { Button } from "../tremor/Button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "../tremor/DropdownMenu";
import { formatValue, toCssId } from "../../lib/render";
import { translate } from "../../lib/translate";

type DropdownProps = {
  label?: string;
  headers: Column[];
  data: (string | number | boolean)[][];
  onChange: (newVars: Record<string, string | string[]>) => void;
  vars?: Record<string, string | string[]>;
  idPrefix: string;
};

function DashboardDropdownMulti ({
  label,
  data,
  headers,
  onChange,
  vars,
  idPrefix,
}: DropdownProps) {
  const valueIndex = headers.findIndex((header) => header.tag === "value");
  const labelIndex = headers.findIndex((header) => header.tag === "label");
  const hintIndex = headers.findIndex((header) => header.tag === "hint");
  const varName = headers[valueIndex].name;
  const selectedVal = (vars ?? {})[varName];
  const selectedValArr = selectedVal
    ? Array.isArray(selectedVal)
      ? selectedVal
      : [selectedVal]
    : [];

  // Determine if all are currently selected
  // If no selectedVal is set (undefined/null), treat as all selected (default state)
  // If selectedVal is empty array, treat as none selected (explicitly unselected)
  // If selectedVal has all values, treat as all selected (explicitly selected all)
  const isDefaultState = selectedVal === undefined || selectedVal === null;
  const hasAllValues = selectedValArr.length === data.length;
  const allSelected = isDefaultState || hasAllValues;
  const noneSelected = !isDefaultState && selectedValArr.length === 0;

  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="ml-2 select-none print:hidden">
      <DropdownMenu open={isOpen} onOpenChange={setIsOpen}>
        <DropdownMenuTrigger asChild>
          <Button
            variant="secondary"
            className="flex w-full items-center justify-between my-1 data-[state=open]:bg-cbga data-[state=open]:dark:bg-dbga"
            id={toCssId(`${idPrefix}${varName}`)}
          >
            {label ?? varName} (
            {noneSelected ? 0 : selectedValArr.length || data.length})
            <SelectPrimitives.Icon asChild>
              <RiExpandUpDownLine className="ml-2 size-4 shrink-0 text-ctext2 dark:text-dtext2" />
            </SelectPrimitives.Icon>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          {data.map((row) => {
            const value = formatValue(
              row[valueIndex],
              headers[valueIndex].type,
            );
            const isChecked =
              isDefaultState || selectedValArr.includes(value.toString());

            return (
              <DropdownMenuCheckboxItem
                key={value}
                checked={isChecked}
                onSelect={(e) => e.preventDefault()}
                onCheckedChange={(checked) => {
                  if (isDefaultState) {
                    // Default state - all selected, clicking deselects this one
                    const allValues = data.map((row) =>
                      formatValue(
                        row[valueIndex],
                        headers[valueIndex].type,
                      ).toString(),
                    );
                    const newSelection = allValues.filter(
                      (v) => v !== value.toString(),
                    );
                    onChange({ ...vars, [varName]: newSelection });
                  } else {
                    // Normal toggle behavior
                    const valSet = new Set(selectedValArr);
                    if (checked) {
                      valSet.add(value.toString());
                    } else {
                      valSet.delete(value.toString());
                    }
                    onChange({ ...vars, [varName]: Array.from(valSet) });
                  }
                }}
                hint={
                  hintIndex !== -1
                    ? formatValue(
                      row[hintIndex],
                      headers[hintIndex].type,
                      true,
                    ).toString()
                    : undefined
                }
              >
                {row[labelIndex !== -1 ? labelIndex : valueIndex]}
              </DropdownMenuCheckboxItem>
            );
          })}
          <div className="border-t mt-1 pt-1">
            <Button
              variant="ghost"
              className="w-full justify-center text-xs"
              onClick={() => {
                if (allSelected && !noneSelected) {
                  // Unselect all - set to empty array
                  onChange({ ...vars, [varName]: [] });
                } else {
                  // Select all - set to all values
                  const allValues = data.map((row) =>
                    formatValue(
                      row[valueIndex],
                      headers[valueIndex].type,
                    ).toString(),
                  );
                  onChange({ ...vars, [varName]: allValues });
                }
              }}
            >
              {allSelected && !noneSelected ? translate("Unselect all") : translate("Select all")}
            </Button>
          </div>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

export default DashboardDropdownMulti;
