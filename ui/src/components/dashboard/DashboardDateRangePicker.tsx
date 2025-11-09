// SPDX-License-Identifier: MPL-2.0

import { Column, Result } from "../../lib/types";
import { DateRangePicker } from "../tremor/DatePicker";
import { Label } from "../tremor/Label";
import { cx, getLocalDate } from "../../lib/utils";
import { translate } from "../../lib/translate";

type PickerProps = {
  label?: string;
  headers: Column[];
  data: Result["sections"][0]["queries"][0]["rows"]
  onChange: (newVars: Record<string, string | string[]>) => void;
  vars?: Record<string, string | string[]>;
};

function DashboardDateRangePicker ({
  label,
  data,
  headers,
  onChange,
  vars,
}: PickerProps) {
  const defaultFromValue = headers.findIndex((header) => header.tag === "defaultFrom");
  if (defaultFromValue === -1) {
    return null;
  }
  const defaultFromValueHeader = headers[defaultFromValue];
  const fromVarName = defaultFromValueHeader.name;
  const fromVarField = (vars ?? {})[fromVarName];
  const fromDefaultValue = data[0][defaultFromValue];
  const fromSelectedDate = Array.isArray(fromVarField) ? fromVarField[0] : fromVarField;

  const toDefaultValueIndex = headers.findIndex((header) => header.tag === "defaultTo");
  if (toDefaultValueIndex === -1) {
    return null;
  }
  const toDefaultValueHeader = headers[toDefaultValueIndex];
  const toVarName = toDefaultValueHeader.name;
  const toVarField = (vars ?? {})[toVarName];
  const toDefaultValue = data[0][toDefaultValueIndex];
  const toSelectedDate = Array.isArray(toVarField) ? toVarField[0] : toVarField;

  const presets = [
    {
      label: translate("Today"),
      dateRange: {
        from: new Date(),
        to: new Date(),
      },
    },
    {
      label: translate("Last 7 days"),
      dateRange: {
        from: new Date(new Date().setDate(new Date().getDate() - 7)),
        to: new Date(),
      },
    },
    {
      label: translate("Last 30 days"),
      dateRange: {
        from: new Date(new Date().setDate(new Date().getDate() - 30)),
        to: new Date(),
      },
    },
    {
      label: translate("Last 3 months"),
      dateRange: {
        from: new Date(new Date().setMonth(new Date().getMonth() - 3)),
        to: new Date(),
      },
    },
    {
      label: translate("Last 6 months"),
      dateRange: {
        from: new Date(new Date().setMonth(new Date().getMonth() - 6)),
        to: new Date(),
      },
    },
    {
      label: translate("Month to date"),
      dateRange: {
        from: new Date(new Date().setDate(1)),
        to: new Date(),
      },
    },
    {
      label: translate("Year to date"),
      dateRange: {
        from: new Date(new Date().setFullYear(new Date().getFullYear(), 0, 1)),
        to: new Date(),
      },
    },
  ];

  return (
    <>
      {label && <Label htmlFor={label} className="ml-3 pr-1 print:hidden">{label}:</Label>}
      <div className={cx("select-none print:hidden", { ["ml-2"]: !label })}>
        <DateRangePicker
          id={label}
          presets={presets}
          enableYearNavigation
          placeholder={translate("Select date range")}
          translations={{
            cancel: translate("Cancel"),
            apply: translate("Apply"),
          }}
          defaultValue={!fromDefaultValue && !toDefaultValue ? undefined : {
            from: typeof fromDefaultValue === "boolean" || !fromDefaultValue ? undefined : getLocalDate(fromDefaultValue),
            to: typeof toDefaultValue === "boolean" || !toDefaultValue ? undefined : getLocalDate(toDefaultValue),
          }}
          value={!fromSelectedDate && !toSelectedDate ? undefined : {
            from: fromSelectedDate ? getLocalDate(fromSelectedDate) : typeof fromDefaultValue === "boolean" || !fromDefaultValue ? undefined : getLocalDate(fromDefaultValue),
            to: toSelectedDate ? getLocalDate(toSelectedDate) : typeof toDefaultValue === "boolean" || !toDefaultValue ? undefined : getLocalDate(toDefaultValue),
          }}
          onChange={(value = { from: undefined, to: undefined }) => {
            const varsCopy = { ...vars };
            if (value.from === undefined) {
              delete varsCopy[fromVarName];
            } else {
              const fromDateString = `${value.from.getFullYear()}-${value.from.toLocaleDateString("en-US", { month: "2-digit" })}-${value.from.toLocaleDateString("en-US", { day: "2-digit" })}`;
              varsCopy[fromVarName] = fromDateString;
            }
            if (value.to === undefined) {
              delete varsCopy[toVarName];
            } else {
              const toDateString = `${value.to.getFullYear()}-${value.to.toLocaleDateString("en-US", { month: "2-digit" })}-${value.to.toLocaleDateString("en-US", { day: "2-digit" })}`;
              varsCopy[toVarName] = toDateString;
            }
            onChange(varsCopy);
          }}
          className={"min-w-40 my-1"}
        />
      </div>
    </>
  );
}

export default DashboardDateRangePicker;
