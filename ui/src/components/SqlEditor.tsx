// SPDX-License-Identifier: MPL-2.0

import { Editor } from "@monaco-editor/react";
import * as monaco from "monaco-editor";
import { useEffect, useRef, useContext } from "react";
import { DarkModeContext } from "../contexts/DarkModeContext";

export function SqlEditor ({
  onChange,
  onRun,
  content,
}: {
  onChange: (value: string | undefined) => void;
  onRun: () => void;
  content: string;
}) {
  const { isDarkMode } = useContext(DarkModeContext);

  // Using a ref so we don't have to recreate the editor if onRun changes
  const runRef = useRef(onRun);

  // We handle this command in monac and outside
  // so even if the editor is not focused the shortcut works
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Check for Ctrl+Enter or Cmd+Enter (Mac)
      if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
        e.preventDefault();
        runRef.current();
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onRun]);

  useEffect(() => {
    runRef.current = onRun;
  }, [onRun]);

  return (
    <Editor
      height="100%"
      defaultLanguage="sql"
      value={content}
      onChange={onChange}
      theme={isDarkMode ? "vs-dark" : "light"}
      options={{
        minimap: { enabled: false },
        fontSize: 14,
        lineNumbers: "on",
        scrollBeyondLastLine: true,
        wordWrap: "on",
        automaticLayout: true,
        formatOnPaste: true,
        formatOnType: true,
        suggestOnTriggerCharacters: true,
        quickSuggestions: true,
        tabSize: 2,
        bracketPairColorization: { enabled: true },
      }}
      onMount={(editor) => {
        editor.addCommand(
          monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter,
          () => {
            runRef.current();
          },
        );
      }}
    />
  );
}
