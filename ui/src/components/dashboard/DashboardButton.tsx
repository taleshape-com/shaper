// SPDX-License-Identifier: MPL-2.0

import { RiFileDownloadLine, RiLoader3Fill } from "@remixicon/react";
import { Column, Result } from "../../lib/types";
import { Button } from "../tremor/Button";
import { Label } from "../tremor/Label";
import { useState } from "react";
import { cx } from "../../lib/utils";
import { toCssId } from "../../lib/render";

type ButtonProps = {
  label?: string;
  headers: Column[];
  data: Result["sections"][0]["queries"][0]["rows"];
  baseUrl?: string;
  getJwt: () => Promise<string>;
  idPrefix: string;
};

function DashboardButton ({
  label,
  data,
  headers,
  baseUrl,
  getJwt,
  idPrefix,
}: ButtonProps) {
  const [isLoading, setIsLoading] = useState(false);

  const handleDownload = async () => {
    setIsLoading(true);
    try {
      const jwt = await getJwt();
      const originalUrl = `${data[0][0]}`;
      const isPdf = originalUrl.includes("/pdf/");

      if (isPdf) {
        const [urlWithoutParams, params] = originalUrl.split("?");
        const parts = urlWithoutParams.split("/");
        // Expected: /api/dashboards/:id/pdf/:filename
        const dashboardId = parts[3];
        const filename = parts[5];

        const response = await fetch(`${baseUrl}/api/dashboards/${dashboardId}/pdf${params ? `?${params}` : ""}`, {
          method: "POST",
          headers: {
            Authorization: jwt,
          },
        });

        if (!response.ok) {
          throw new Error("Failed to request PDF");
        }

        const { key } = await response.json();
        const downloadUrl = `${baseUrl}/api/download/${key}/${filename}`;

        const link = document.createElement("a");
        link.href = downloadUrl;
        link.download = filename;
        document.body.appendChild(link);
        link.click();
        link.remove();
      } else {
        const url = `${baseUrl}${originalUrl}`;
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
        link.download = url.split("#")[0].split("?")[0].split("/").pop() ?? "download";
        document.body.appendChild(link);
        link.click();
        link.remove();
      }
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
        className={cx("my-1 select-none print:hidden", {
          "ml-2": !label,
        })}
        id={toCssId(`${idPrefix}${label ? `${label}-` : ""}${headers[0].name}`)}
      >
        <span className="flex items-center justify-between">
          {headers[0].name}
          {isLoading ? (
            <RiLoader3Fill className="ml-1.5 size-4 text-ctext2 dark:text-dtext2 animate-spin" />
          ) : (
            <RiFileDownloadLine className="ml-1.5 size-4 text-ctext2 dark:text-dtext2" />
          )}
        </span>
      </Button >
    </>
  );
}

export default DashboardButton;
