// SPDX-License-Identifier: MPL-2.0

import { loader } from "@monaco-editor/react";
import * as monaco from "monaco-editor";

import editorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker";

// Initialize Monaco Editor from source files instead of CDN
self.MonacoEnvironment = {
  getWorker () {
    return new editorWorker();
  },
};
loader.config({ monaco });
loader.init();
