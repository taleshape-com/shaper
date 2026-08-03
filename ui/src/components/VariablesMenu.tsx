// SPDX-License-Identifier: MPL-2.0

import { useState } from "react";
import {
  RiAddLine,
  RiDeleteBinLine,
  RiCloseLine,
  RiEqualizerLine,
  RiEditLine,
} from "@remixicon/react";
import { cx } from "../lib/utils";
import { useAuth, Variables } from "../lib/auth";
import { Input } from "./tremor/Input";
import { Button } from "./tremor/Button";
import { Tooltip } from "./tremor/Tooltip";
import {
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "./tremor/Dialog";

interface VariableItem {
  id: string;
  key: string;
  type: "string" | "array";
  stringValue: string;
  arrayValues: string[];
}

interface VariablesMenuProps {
  onVariablesChange?: () => void;
}

function generateId (): string {
  return Math.random().toString(36).substring(2, 9);
}

function variablesToItems (vars: Variables): VariableItem[] {
  return Object.entries(vars).map(([key, value]) => {
    if (Array.isArray(value)) {
      return {
        id: generateId(),
        key,
        type: "array",
        stringValue: "",
        arrayValues: value.length > 0 ? [...value] : [""],
      };
    }
    return {
      id: generateId(),
      key,
      type: "string",
      stringValue: String(value),
      arrayValues: [""],
    };
  });
}

function itemsToVariables (items: VariableItem[]): Variables {
  const result: Variables = {};
  for (const item of items) {
    const trimmedKey = item.key.trim();
    if (!trimmedKey) continue;
    if (item.type === "string") {
      result[trimmedKey] = item.stringValue;
    } else {
      result[trimmedKey] = item.arrayValues;
    }
  }
  return result;
}

export function VariablesMenu ({ onVariablesChange }: VariablesMenuProps) {
  const auth = useAuth();
  const [isOpen, setIsOpen] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [items, setItems] = useState<VariableItem[]>(() =>
    variablesToItems(auth.variables),
  );

  const handleOpenChange = (open: boolean) => {
    if (open) {
      setItems(variablesToItems(auth.variables));
    }
    setIsOpen(open);
  };

  const handleKeyChange = (id: string, newKey: string) => {
    setItems((prev) =>
      prev.map((item) => (item.id === id ? { ...item, key: newKey } : item)),
    );
  };

  const handleTypeChange = (id: string, newType: "string" | "array") => {
    setItems((prev) =>
      prev.map((item) => {
        if (item.id !== id) return item;
        if (newType === "array") {
          const initArray = item.stringValue ? [item.stringValue] : [""];
          return { ...item, type: newType, arrayValues: initArray };
        } else {
          const initString =
            item.arrayValues.length > 0 ? item.arrayValues[0] : "";
          return { ...item, type: newType, stringValue: initString };
        }
      }),
    );
  };

  const handleStringValueChange = (id: string, val: string) => {
    setItems((prev) =>
      prev.map((item) =>
        item.id === id ? { ...item, stringValue: val } : item,
      ),
    );
  };

  const handleArrayValueChange = (
    id: string,
    index: number,
    val: string,
  ) => {
    setItems((prev) =>
      prev.map((item) => {
        if (item.id !== id) return item;
        const newArr = [...item.arrayValues];
        newArr[index] = val;
        return { ...item, arrayValues: newArr };
      }),
    );
  };

  const handleAddArrayItem = (id: string) => {
    setItems((prev) =>
      prev.map((item) => {
        if (item.id !== id) return item;
        return { ...item, arrayValues: [...item.arrayValues, ""] };
      }),
    );
  };

  const handleRemoveArrayItem = (id: string, index: number) => {
    setItems((prev) =>
      prev.map((item) => {
        if (item.id !== id) return item;
        const newArr = item.arrayValues.filter((_, i) => i !== index);
        return {
          ...item,
          arrayValues: newArr.length > 0 ? newArr : [""],
        };
      }),
    );
  };

  const handleAddVariable = () => {
    const newItem: VariableItem = {
      id: generateId(),
      key: "",
      type: "string",
      stringValue: "",
      arrayValues: [""],
    };
    setItems((prev) => [...prev, newItem]);
  };

  const handleRemoveVariable = (id: string) => {
    setItems((prev) => prev.filter((item) => item.id !== id));
  };

  const handleSave = async () => {
    setIsSaving(true);
    const vars = itemsToVariables(items);
    const ok = await auth.updateVariables(JSON.stringify(vars));
    setIsSaving(false);
    if (ok) {
      if (onVariablesChange) {
        onVariablesChange();
      }
      setIsOpen(false);
    }
  };

  const handleCancel = () => {
    setIsOpen(false);
  };

  const entries = Object.entries(auth.variables);
  const activeCount = entries.length;

  const tooltipContent =
    entries.length === 0 ? (
      <span className="italic opacity-80 text-xs">No variables set</span>
    ) : (
      <div className="font-mono text-xs space-y-1 max-w-xs overflow-hidden">
        {entries.map(([k, v]) => (
          <div key={k} className="truncate">
            <span className="font-semibold opacity-90">{k}: </span>
            <span className="opacity-80">
              {Array.isArray(v)
                ? `[${v.map((item) => JSON.stringify(item)).join(", ")}]`
                : JSON.stringify(v)}
            </span>
          </div>
        ))}
      </div>
    );

  return (
    <div className="mt-5 px-4 w-full">
      <div className="text-sm font-medium font-display mb-2 block">
        Variables
      </div>

      <Dialog open={isOpen} onOpenChange={handleOpenChange}>
        <Tooltip content={tooltipContent} side="right" sideOffset={8}>
          <DialogTrigger asChild>
            <button
              type="button"
              className="w-full flex items-center justify-between px-3 py-2 bg-cbgs dark:bg-dbgs hover:bg-cbga dark:hover:bg-dbga border border-cb dark:border-db rounded-md text-xs font-medium text-ctext dark:text-dtext transition-colors shadow-sm"
            >
              <span className="flex items-center space-x-2">
                <RiEqualizerLine className="size-4 text-ctext2 dark:text-dtext2" />
                <span>
                  {activeCount > 0
                    ? `${activeCount} active variable${activeCount > 1 ? "s" : ""}`
                    : "Set Variables"}
                </span>
              </span>
              <RiEditLine className="size-3.5 text-ctext2 dark:text-dtext2" />
            </button>
          </DialogTrigger>
        </Tooltip>

        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Variables</DialogTitle>
            <DialogDescription>
              Set variables to simulate parameters passed when embedding a dashboard in another application.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3 my-2 max-h-[60vh] overflow-y-auto pr-1">
            {items.length === 0 ? (
              <div className="text-xs text-ctext2 dark:text-dtext2 italic py-4 text-center">
                No variables defined.
              </div>
            ) : (
              items.map((item) => (
                <div
                  key={item.id}
                  className="p-3 bg-cbgs dark:bg-dbgs border border-cb dark:border-db rounded-md space-y-2 text-xs shadow-sm"
                >
                  {/* Header Row: Key + Type Selector + Delete */}
                  <div className="flex items-center space-x-2">
                    <Input
                      placeholder="Variable key"
                      value={item.key}
                      onChange={(e) => handleKeyChange(item.id, e.target.value)}
                      className="flex-grow text-xs py-1"
                    />
                    <div className="flex border border-cb dark:border-db rounded-md overflow-hidden shrink-0">
                      <button
                        type="button"
                        className={cx(
                          "px-2.5 py-1 text-xs font-medium transition-colors",
                          item.type === "string"
                            ? "bg-cbg dark:bg-dbg text-ctext dark:text-dtext font-semibold"
                            : "text-ctext2 dark:text-dtext2 hover:text-ctext dark:hover:text-dtext",
                        )}
                        onClick={() => handleTypeChange(item.id, "string")}
                      >
                        String
                      </button>
                      <button
                        type="button"
                        className={cx(
                          "px-2.5 py-1 text-xs font-medium transition-colors border-l border-cb dark:border-db",
                          item.type === "array"
                            ? "bg-cbg dark:bg-dbg text-ctext dark:text-dtext font-semibold"
                            : "text-ctext2 dark:text-dtext2 hover:text-ctext dark:hover:text-dtext",
                        )}
                        onClick={() => handleTypeChange(item.id, "array")}
                      >
                        Array
                      </button>
                    </div>
                    <button
                      type="button"
                      onClick={() => handleRemoveVariable(item.id)}
                      className="text-ctext2 hover:text-cerr dark:text-dtext2 dark:hover:text-derr p-1 rounded shrink-0 transition-colors"
                      title="Delete variable"
                    >
                      <RiDeleteBinLine className="size-4" />
                    </button>
                  </div>

                  {/* Value Section */}
                  {item.type === "string" ? (
                    <Input
                      placeholder="Value"
                      value={item.stringValue}
                      onChange={(e) =>
                        handleStringValueChange(item.id, e.target.value)
                      }
                      className="w-full text-xs py-1"
                    />
                  ) : (
                    <div className="space-y-1.5 pl-1">
                      {item.arrayValues.map((arrVal, idx) => (
                        <div
                          key={idx}
                          className="flex items-center space-x-1.5"
                        >
                          <Input
                            placeholder={`Value ${idx + 1}`}
                            value={arrVal}
                            onChange={(e) =>
                              handleArrayValueChange(
                                item.id,
                                idx,
                                e.target.value,
                              )
                            }
                            className="flex-grow text-xs py-1"
                          />
                          <button
                            type="button"
                            onClick={() => handleRemoveArrayItem(item.id, idx)}
                            className="text-ctext2 hover:text-cerr dark:text-dtext2 dark:hover:text-derr p-1 rounded shrink-0 transition-colors"
                            title="Remove item"
                          >
                            <RiCloseLine className="size-3.5" />
                          </button>
                        </div>
                      ))}
                      <button
                        type="button"
                        onClick={() => handleAddArrayItem(item.id)}
                        className="text-xs text-ctext2 dark:text-dtext2 hover:text-ctext dark:hover:text-dtext flex items-center space-x-1 pt-0.5 transition-colors font-medium"
                      >
                        <RiAddLine className="size-3.5" />
                        <span>Add item</span>
                      </button>
                    </div>
                  )}
                </div>
              ))
            )}

            <Button
              type="button"
              variant="secondary"
              className="w-full py-1.5 text-xs font-medium flex items-center justify-center space-x-1"
              onClick={handleAddVariable}
            >
              <RiAddLine className="size-3.5 inline" />
              <span>Add Variable</span>
            </Button>
          </div>

          <DialogFooter className="mt-4 flex items-center justify-end space-x-2">
            <Button
              type="button"
              variant="secondary"
              className="py-1.5 px-3 text-xs font-medium"
              onClick={handleCancel}
              disabled={isSaving}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="primary"
              className="py-1.5 px-3 text-xs font-medium"
              onClick={handleSave}
              disabled={isSaving}
            >
              {isSaving ? "Saving..." : "Save"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
