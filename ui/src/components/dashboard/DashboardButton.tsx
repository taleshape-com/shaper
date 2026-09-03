// SPDX-License-Identifier: MPL-2.0

import {
  RiCodeSSlashLine,
  RiFileDownloadLine,
  RiFileExcel2Line,
  RiFilePdf2Line,
  RiFileTextLine,
  RiLoader3Fill,
} from "@remixicon/react";
import { Column, Result } from "../../lib/types";
import { Button } from "../tremor/Button";
import { Label } from "../tremor/Label";
import { useState } from "react";
import { cx } from "../../lib/utils";
import { toCssId } from "../../lib/render";
import { useToast } from "../../hooks/useToast";

type ButtonProps = {
  label?: string;
  headers: Column[];
  data: Result["sections"][0]["queries"][0]["rows"];
  baseUrl?: string;
  getJwt: () => Promise<string>;
  idPrefix: string;
};

function getFilenameFromHeader (response: Response, fallback: string): string {
  const disposition = response.headers.get("content-disposition");
  if (disposition) {
    const match = disposition.match(/filename=["']?([^"';]+)["']?/i);
    if (match && match[1]) {
      return match[1];
    }
  }
  return fallback;
}

function DashboardButton ({
  label,
  data,
  headers,
  baseUrl = "",
  getJwt,
  idPrefix,
}: ButtonProps) {
  const [isLoading, setIsLoading] = useState(false);
  const { toast } = useToast();

  const relativeUrl = data[0][0] as string;
  const urlPath = relativeUrl.split("?")[0];
  const defaultFilename = urlPath.split("/").pop() || "download";
  const extension = defaultFilename.split(".").pop()?.toLowerCase();

  const handleDownload = async () => {
    setIsLoading(true);
    let writableStream: WritableStream | null = null;

    try {
      const isStreamableLargeFile = extension === "csv" || extension === "json";

      // For large streamable files (CSV & JSON) on Chromium browsers,
      // prompt the Save As dialog immediately during the user click gesture
      if (isStreamableLargeFile && "showSaveFilePicker" in window) {
        const pickerOptions: {
          suggestedName: string;
          types?: { description: string; accept: Record<string, string[]> }[];
        } = {
          suggestedName: defaultFilename,
        };

        if (extension === "csv") {
          pickerOptions.types = [
            {
              description: "CSV file",
              accept: { "text/csv": [".csv"] },
            },
          ];
        } else if (extension === "json") {
          pickerOptions.types = [
            {
              description: "JSON file",
              accept: { "application/json": [".json"] },
            },
          ];
        }

        try {

          const fileHandle = await (window as any).showSaveFilePicker(pickerOptions);
          writableStream = await fileHandle.createWritable();
        } catch (err: unknown) {
          if ((err as DOMException)?.name === "AbortError") {
            // User cancelled the file picker dialog
            setIsLoading(false);
            return;
          }
          throw err;
        }
      }

      const jwt = await getJwt();
      const response = await fetch(`${baseUrl}${relativeUrl}`, {
        headers: {
          Authorization: jwt,
        },
      });

      if (!response.ok) {
        if (writableStream) {
          try {
            await writableStream.abort("Download failed on server");
          } catch {
            // Ignore abort error
          }
        }
        const json = await response.json().catch(() => ({ error: response.statusText }));
        throw new Error(json.error || `Download failed: ${response.statusText}`);
      }

      if (writableStream && response.body) {
        // Stream directly to disk
        await response.body.pipeTo(writableStream);
      } else {
        // Buffer to Blob (PDF, XLSX, PNG, and Safari/Firefox fallback for CSV/JSON)
        const blob = await response.blob();
        const filename = getFilenameFromHeader(response, defaultFilename);
        const downloadUrl = URL.createObjectURL(blob);
        const link = document.createElement("a");
        link.href = downloadUrl;
        link.download = filename;
        document.body.appendChild(link);
        link.click();
        link.remove();
        URL.revokeObjectURL(downloadUrl);
      }
    } catch (error: unknown) {
      if ((error as DOMException)?.name !== "AbortError") {
        console.error("Download error:", error);
        toast({
          title: "Download failed",
          description: (error as Error).message || "An unexpected error occurred during download.",
          variant: "error",
        });
      }
    } finally {
      setTimeout(() => {
        setIsLoading(false);
      }, 500);
    }
  };

  return (
    <div className="flex items-center print:hidden">
      {label && (
        <Label className="ml-3 pr-1 shrink-0">
          {label}:
        </Label>
      )}
      <Button
        onClick={handleDownload}
        disabled={isLoading}
        variant="secondary"
        className={cx("my-1 select-none", {
          "ml-2": !label,
        })}
        id={toCssId(`${idPrefix}${label ? `${label}-` : ""}${headers[0].name}`)}
      >
        <span className="flex items-center justify-between">
          {headers[0].name}
          {isLoading ? (
            <RiLoader3Fill className="ml-1.5 size-4 text-ctext2 dark:text-dtext2 animate-spin" />
          ) : extension === "csv" ? (
            <RiFileTextLine className="ml-1.5 size-4 text-ctext2 dark:text-dtext2" />
          ) : extension === "xlsx" ? (
            <RiFileExcel2Line className="ml-1.5 size-4 text-ctext2 dark:text-dtext2" />
          ) : extension === "pdf" ? (
            <RiFilePdf2Line className="ml-1.5 size-4 text-ctext2 dark:text-dtext2" />
          ) : extension === "json" ? (
            <RiCodeSSlashLine className="ml-1.5 size-4 text-ctext2 dark:text-dtext2" />
          ) : (
            <RiFileDownloadLine className="ml-1.5 size-4 text-ctext2 dark:text-dtext2" />
          )}
        </span>
      </Button >
    </div>
  );
}

export default DashboardButton;
