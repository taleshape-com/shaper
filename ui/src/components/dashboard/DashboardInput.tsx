// SPDX-License-Identifier: MPL-2.0

import { Column } from "../../lib/types";
import { formatValue } from "../../lib/render";
import { Input } from "../tremor/Input";
import { Label } from "../tremor/Label";
import { cx } from "../../lib/utils";
import { useThrottledCallback } from "use-debounce";
import { useState, useEffect } from "react";

type InputProps = {
  label?: string;
  headers: Column[];
  data: (string | number | boolean)[][];
  onChange: (newVars: Record<string, string | string[]>) => void;
  vars?: Record<string, string | string[]>;
};

function DashboardInput ({
  label,
  data,
  headers,
  onChange,
  vars,
}: InputProps) {
  const valueIndex = headers.findIndex((header) => header.tag === "hint");
  const varName = headers[valueIndex].name;
  const varField = (vars ?? {})[varName];
  const externalValue = Array.isArray(varField) ? varField[0] : varField;

  // Local state for immediate UI updates
  const [localValue, setLocalValue] = useState(externalValue || "");

  // Sync local state when external value changes
  useEffect(() => {
    setLocalValue(externalValue || "");
  }, [externalValue]);

  // Get placeholder from data if available
  const placeholder = data.length > 0 ?
    formatValue(data[0][valueIndex], headers[valueIndex].type).toString() :
    "";

  // Throttled onChange handler - calls at most every 300ms while user is typing
  const throttledOnChange = useThrottledCallback((value: string) => {
    onChange({ ...vars, [varName]: value });
  }, 500);

  return (
    <>
      {label && (
        <Label htmlFor={label} className="ml-3 pr-1 print:hidden">
          {label}:
        </Label>
      )}
      <div className={cx("print:hidden", { ["ml-2"]: !label })}>
        <Input
          id={label}
          value={localValue}
          placeholder={placeholder}
          onChange={(e) => {
            const newValue = e.target.value;
            // Update local state immediately for responsive UI
            setLocalValue(newValue);
            // Throttle the actual API call
            throttledOnChange(newValue);
          }}
          className="mx-auto my-0 min-w-[120px]"
        />
      </div>
    </>
  );
}

export default DashboardInput;
