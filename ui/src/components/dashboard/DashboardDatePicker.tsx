// SPDX-License-Identifier: MPL-2.0

import { Column } from "../../lib/types";
import { DatePicker } from "../tremor/DatePicker";
import { Label } from "../tremor/Label";
import { cx, getLocalDate } from "../../lib/utils";
import { translate } from "../../lib/translate";
import { toCssId } from "../../lib/render";

type PickerProps = {
  label?: string;
  headers: Column[];
  data: (string | number | boolean)[][];
  onChange: (newVars: Record<string, string | string[]>) => void;
  vars?: Record<string, string | string[]>;
  idPrefix: string;
};

function DashboardDatePicker ({
  label,
  data,
  headers,
  onChange,
  vars,
  idPrefix,
}: PickerProps) {
  const defaultValueIndex = headers.findIndex((header) => header.tag === "default");
  if (defaultValueIndex === -1) {
    return null;
  }
  const defaultValueHeader = headers[defaultValueIndex];
  const varName = defaultValueHeader.name;
  const varField = (vars ?? {})[varName];
  const defaultValue = data[0][defaultValueIndex];
  const selectedDate = Array.isArray(varField) ? varField[0] : varField;

  return (
    <>
      {label && <Label htmlFor={label} className="ml-3 pr-1 print:hidden">{label}:</Label>}
      <div className={cx("select-none print:hidden", { ["ml-2"]: !label })}>
        <DatePicker
          id={toCssId(`${idPrefix}${varName}`)}
          defaultValue={typeof defaultValue === "boolean" || !defaultValue ? undefined : getLocalDate(defaultValue)}
          enableYearNavigation
          value={selectedDate ? getLocalDate(selectedDate) : undefined}
          placeholder={translate("Select date")}
          onChange={value => {
            if (!value) {
              if (!vars) {
                return;
              }
              const varsCopy = { ...vars };
              delete varsCopy[varName];
              onChange(varsCopy);
              return;
            }
            const dateString = `${value.getFullYear()}-${value.toLocaleDateString("en-US", { month: "2-digit" })}-${value.toLocaleDateString("en-US", { day: "2-digit" })}`;
            onChange({ ...vars, [varName]: dateString });
          }}
          className={"min-w-28 my-1"}
        />
      </div>
    </>
  );
}

export default DashboardDatePicker;
