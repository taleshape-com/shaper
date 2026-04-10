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

type ButtonProps = {
  label?: string;
  headers: Column[];
  data: Result["sections"][0]["queries"][0]["rows"];
  baseUrl?: string;
  getJwt: () => Promise<string>;
  idPrefix: string;
};

const getIcon = (url: string) => {
  const extension = url.split("?")[0].split(".").pop()?.toLowerCase();
  switch (extension) {
  case "csv":
    return RiFileTextLine;
  case "xlsx":
    return RiFileExcel2Line;
  case "pdf":
    return RiFilePdf2Line;
  case "json":
    return RiCodeSSlashLine;
  default:
    return RiFileDownloadLine;
  }
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

  const relativeUrl = data[0][0] as string;
  const Icon = getIcon(relativeUrl);

  const handleDownload = async () => {
    setIsLoading(true);
    try {
      const jwt = await getJwt();
      const separator = relativeUrl.includes("?") ? "&" : "?";
      const response = await fetch(`${baseUrl}${relativeUrl}${separator}mode=url`, {
        headers: {
          "Content-Type": "application/json",
          Authorization: jwt,
        },
      });
      const json = await response.json();
      if (!response.ok) {
        throw new Error("Download request failed: " + (json.error || response.statusText));
      }
      const downloadUrl = (json.url as string).startsWith("http://") || (json.url as string).startsWith("https://")
        ? json.url
        : `${baseUrl?.endsWith("/") ? baseUrl.substring(0, baseUrl.length - 1) : baseUrl}${json.url}`;
      const filename = downloadUrl.split("/").pop() || "download";

      const link = document.createElement("a");
      link.href = downloadUrl;
      link.download = filename;
      document.body.appendChild(link);
      link.click();
      link.remove();
    } catch (error) {
      // TODO: Handle error (e.g., show an error message to the user)
      //       We either have to switch to buffering in memory again or we change the server so the file is rendered and cached before the download link is returned.
      //       The problem with that is that it defeats the purpose of streaming the file. Then we have to wait on the server side until the file is rendered,
      //       and we probably slow things down because we have to write the file to disk on the server which I don't want to do - especially for large files.
      //       One thing to consider is to buffer in JS, but show the user a progress bar in the app (instead of the browsers own functionality).
      console.error("Download error:", error);
    } finally {
      setTimeout(() => {
        setIsLoading(false);
      }, 500);
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
            <Icon className="ml-1.5 size-4 text-ctext2 dark:text-dtext2" />
          )}
        </span>
      </Button >
    </>
  );
}

export default DashboardButton;
