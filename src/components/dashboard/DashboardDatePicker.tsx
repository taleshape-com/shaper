import { Column, Result } from "../../lib/dashboard";
import { DatePicker } from "../tremor/DatePicker";
import { Label } from "../tremor/Label"
import { cx } from "../../lib/utils";

type PickerProps = {
  label?: string;
  headers: Column[];
  data: Result['sections'][0]['queries'][0]['rows']
  onChange: (newVars: Record<string, string | string[]>) => void;
  vars?: Record<string, string | string[]>;
};

const translations: Record<string, Record<string, string>> = {
  'Select date': {
    'de': 'Datum wählen'
  }
}
function translate(s: string) {
  const available = translations[s] ?? {}
  for (const lang of navigator.languages) {
    if (lang === 'en' || lang === 'en-US') {
      return s
    }
    const t = available[lang]
    if (t) {
      return t
    }
  }
  return s
}

function DashboardDatePicker({
  label,
  data,
  headers,
  onChange,
  vars,
}: PickerProps) {
  const defaultValueIndex = headers.findIndex((header) => header.tag === "default");
  if (defaultValueIndex === -1) {
    return null;
  }
  const defaultValueHeader = headers[defaultValueIndex];
  const varName = defaultValueHeader.name;
  const varField = (vars ?? {})[varName]
  const defaultValue = data[0][defaultValueIndex]
  const selectedDate = Array.isArray(varField) ? varField[0] : varField

  return (
    <>
      {label && <Label htmlFor={label} className="ml-2 pr-1">{label}:</Label>}
      <div className={cx({ ["ml-2"]: !label })}>
        <DatePicker
          id={label}
          defaultValue={typeof defaultValue === 'boolean' || !defaultValue ? undefined : new Date(defaultValue)}
          enableYearNavigation
          value={selectedDate ? new Date(selectedDate) : undefined}
          placeholder={translate('Select date')}
          onChange={value => {
            if (!value) {
              if (!vars) {
                return
              }
              const varsCopy = { ...vars }
              delete varsCopy[varName]
              onChange(varsCopy);
              return
            }
            const dateString = `${value.getFullYear()}-${value.toLocaleDateString([], { month: "2-digit" })}-${value.toLocaleDateString([], { day: "2-digit" })}`
            onChange({ ...vars, [varName]: dateString });
          }}
          className={"min-w-28 my-1"}
        />
      </div>
    </>
  );
}

export default DashboardDatePicker;


