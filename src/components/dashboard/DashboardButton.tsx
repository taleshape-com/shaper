import * as SelectPrimitives from "@radix-ui/react-select";
import { RiFileDownloadLine } from "@remixicon/react";
import { Column, Result } from "../../lib/dashboard";
import { Button } from "../tremor/Button";
import { formatValue } from "../../lib/render";

type ButtonProps = {
  label?: string;
  headers: Column[];
  data?: Result['sections'][0]['queries'][0]['rows']
};

// TODO: Support multiple buttons in one select to download different file formats
function DashboardButton({
  label,
  data,
  headers,
}: ButtonProps) {
  return (
    <div className="ml-4">
      <a href={formatValue((data ?? [])[0][0]).toString()} download>
        <Button variant="secondary" className="font-normal flex w-full items-center justify-between">
          {label}
          {headers[0].name}
          <SelectPrimitives.Icon asChild>
            <RiFileDownloadLine
              className="ml-2 size-4 shrink-0 text-gray-400 dark:text-gray-600"
            />
          </SelectPrimitives.Icon>
        </Button>
      </a>
    </div>
  );
}

export default DashboardButton;

