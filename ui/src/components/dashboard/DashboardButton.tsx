// SPDX-License-Identifier: MPL-2.0

import * as SelectPrimitives from "@radix-ui/react-select";
import { RiFileDownloadLine } from "@remixicon/react";
import { Column, Result } from "../../lib/types";
import { Button } from "../tremor/Button";
import { Label } from "../tremor/Label";
import { useState } from "react";
import { cx } from "../../lib/utils";

type ButtonProps = {
  label?: string;
  headers: Column[];
  data: Result["sections"][0]["queries"][0]["rows"];
  baseUrl?: string;
  getJwt: () => Promise<string>;
};

function DashboardButton({
  label,
  data,
  headers,
  baseUrl,
  getJwt,
}: ButtonProps) {
  const [isLoading, setIsLoading] = useState(false);

  const handleDownload = async () => {
    setIsLoading(true);
    try {
      const jwt = await getJwt();
      const url = `${baseUrl}${data[0][0]}`;
      const response = await fetch(url, {
        headers: {
          Authorization: jwt,
        },
      });

      if (!response.ok) {
        throw new Error("Download failed");
      }

      const blob = await response.blob();
      const downloadUrl = window.URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = downloadUrl;
      link.download = url.split('#')[0].split('?')[0].split('/').pop() ?? 'download';
      document.body.appendChild(link);
      link.click();
      link.remove();
    } catch (error) {
      console.error("Download error:", error);
      // Handle error (e.g., show an error message to the user)
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      {label && (
        <Label className="ml-3 pr-1 print:hidden">
          {label}:
        </Label>
      )}
      <Button
        onClick={handleDownload}
        disabled={isLoading}
        variant="secondary"
        className={cx("flex items-center justify-between my-1 select-none  print:hidden", {
          "ml-2": !label,
        })}
      >
        {headers[0].name}
        <SelectPrimitives.Icon asChild>
          <RiFileDownloadLine className="ml-2 size-4 shrink-0 text-ctext2 dark:text-dtext2" />
        </SelectPrimitives.Icon>
      </Button >
    </>
  );
}

export default DashboardButton;
