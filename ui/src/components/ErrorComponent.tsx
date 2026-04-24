// SPDX-License-Identifier: MPL-2.0

import React from "react";
import { RiFileCopyLine, RiCheckLine } from "@remixicon/react";
import { copyToClipboard } from "../lib/utils";
import { Button } from "./tremor/Button";

interface ErrorComponentProps {
  error: any;
}

export function ErrorComponent ({ error }: ErrorComponentProps) {
  const [copied, setCopied] = React.useState(false);

  const handleCopy = async () => {
    const textToCopy = error instanceof Error ? error.message : String(error);
    const success = await copyToClipboard(textToCopy);
    if (success) {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div className="flex flex-col items-center justify-center p-8 text-center">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-50">
        Something went wrong
      </h2>
      <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
        {error instanceof Error ? error.message : String(error)}
      </p>
      <div className="mt-6 flex items-center gap-3">
        <Button
          variant="secondary"
          onClick={() => window.location.reload()}
        >
          Reload page
        </Button>
        <Button
          variant="light"
          onClick={handleCopy}
          className="flex items-center gap-2"
        >
          {copied ? (
            <>
              <RiCheckLine className="size-4 text-emerald-500" />
              Copied
            </>
          ) : (
            <>
              <RiFileCopyLine className="size-4" />
              Copy error
            </>
          )}
        </Button>
      </div>
    </div>
  );
}
