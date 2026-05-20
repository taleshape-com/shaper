// SPDX-License-Identifier: MPL-2.0

import React from "react";
import { RiFileCopyLine, RiCheckLine } from "@remixicon/react";
import { copyToClipboard } from "../lib/utils";

export function PreviewError ({ children }: { children: React.ReactNode }) {
  const [copied, setCopied] = React.useState(false);

  const handleCopy = async () => {
    // Extract text from children if it's a string or try to get textContent
    const textToCopy = typeof children === "string" ? children : document.getElementById("preview-error-content")?.textContent || "";
    if (textToCopy) {
      const success = await copyToClipboard(textToCopy);
      if (success) {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      }
    }
  };

  return (
    <div className="fixed w-full h-full p-4 z-50 backdrop-blur-sm flex justify-center">
      <div className="p-4 bg-red-100 text-red-700 rounded mt-32 h-fit flex items-start gap-4">
        <div id="preview-error-content">{children}</div>
        <button
          onClick={handleCopy}
          className="shrink-0 text-red-500 hover:text-red-700 transition-colors"
          title="Copy error message"
        >
          {copied ? (
            <RiCheckLine className="size-5" />
          ) : (
            <RiFileCopyLine className="size-5" />
          )}
        </button>
      </div>
    </div>
  );
}
