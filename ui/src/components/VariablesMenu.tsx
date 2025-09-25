// SPDX-License-Identifier: MPL-2.0

import { useState } from 'react'
import { useDebouncedCallback } from "use-debounce";
import { cx, focusRing, hasErrorInput } from "../lib/utils";
import { useAuth } from "../lib/auth";

interface VariablesMenuProps {
  onVariablesChange?: () => void;
}

const rows = 4;

export function VariablesMenu({ onVariablesChange }: VariablesMenuProps) {
  const [hasVariableError, setHasVariableError] = useState(false);
  const auth = useAuth();

  const onVariablesEdit = useDebouncedCallback((value: string) => {
    auth.updateVariables(value).then(
      (ok) => {
        setHasVariableError(!ok);
        if (ok && onVariablesChange) {
          onVariablesChange();
        }
      },
      () => {
        setHasVariableError(true);
      },
    );
  }, 500);

  return (
    <div className="mt-6 px-4 w-full">
      <label>
        <span className="text-lg font-medium font-display ml-1 mb-2 block">Variables</span>
        <textarea
          className={cx(
            "w-full px-3 py-1.5 bg-cbg dark:bg-dbg text-sm border border-cb dark:border-db shadow-sm outline-none ring-0 rounded-md font-mono resize-none",
            focusRing,
            hasVariableError && hasErrorInput,
          )}
          onChange={(event) => {
            onVariablesEdit(event.target.value);
          }}
          defaultValue={JSON.stringify(auth.variables, null, 2)}
          rows={rows}
        ></textarea>
      </label>
    </div>
  );
}