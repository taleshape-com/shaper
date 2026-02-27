// SPDX-License-Identifier: MPL-2.0

import React from "react";
import { RiDownload2Line } from "@remixicon/react";
import { cx } from "../../lib/utils";
import { translate } from "../../lib/translate";
import { Column } from "../../lib/types";

interface TableDownloadButtonProps {
  headers: Column[];
  data: (string | number | boolean)[][];
  label?: string;
  className?: string;
  id?: string;
}

function escapeCSV (value: string | number | boolean): string {
  const str = String(value);
  if (str.includes(",") || str.includes("\"") || str.includes("\n")) {
    return `"${str.replace(/"/g, "\"\"")}"`;
  }
  return str;
}

export const TableDownloadButton: React.FC<TableDownloadButtonProps> = ({
  headers,
  data,
  label,
  className,
  id,
}) => {
  const handleDownload = React.useCallback(() => {
    const rows = [
      headers.map((h) => escapeCSV(h.name)),
      ...data.map((row) => row.map(escapeCSV)),
    ];
    const csv = rows.map((row) => row.join(",")).join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${label ?? "table"}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }, [headers, data, label]);

  return (
    <div className="absolute inset-0 pointer-events-none print:hidden">
      <button
        className={cx(
          "absolute top-2 right-2 z-50",
          "p-1.5 rounded-md",
          "bg-cbg dark:bg-dbg",
          "border border-cb dark:border-db",
          "text-ctext dark:text-dtext",
          "hover:bg-cbgs dark:hover:bg-dbgs",
          "transition-all duration-100",
          "opacity-0 group-hover:opacity-100",
          "focus:outline-none focus:ring-2 focus:ring-cprimary dark:focus:ring-dprimary",
          "pointer-events-auto",
          className,
        )}
        onClick={handleDownload}
        title={translate("Save as CSV")}
        id={id}
      >
        <RiDownload2Line className="size-4" />
      </button>
    </div>
  );
};
