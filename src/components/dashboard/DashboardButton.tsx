import * as SelectPrimitives from "@radix-ui/react-select";
import { RiFileDownloadLine } from "@remixicon/react";
import { Column, Result } from "../../lib/dashboard";
import { Button } from "../tremor/Button";

type ButtonProps = {
  label?: string;
  headers: Column[];
  data: Result['sections'][0]['queries'][0]['rows']
  searchParams: string;
  baseUrl?: string;
};

// TODO: Support multiple buttons in one select to download different file formats
function DashboardButton({
  label,
  data,
  headers,
  searchParams,
  baseUrl,
}: ButtonProps) {
  return (
    <div className="ml-2">
      <Button
        asChild
        variant="secondary"
        className="font-normal flex w-full items-center justify-between my-1">
        <a href={`${baseUrl}${data[0][0]}?${searchParams}`} download>
          {label}
          {headers[0].name}
          <SelectPrimitives.Icon asChild>
            <RiFileDownloadLine
              className="ml-2 size-4 shrink-0 text-ctext2 dark:text-dtext2"
            />
          </SelectPrimitives.Icon>
        </a>
      </Button>
    </div>
  );
}

export default DashboardButton;

