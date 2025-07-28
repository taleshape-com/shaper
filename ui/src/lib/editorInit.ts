import { loader } from "@monaco-editor/react";
import * as monaco from "monaco-editor";

// Initialize Monaco Editor from source files instead of CDN
self.MonacoEnvironment = {
  getWorker() {
    return new Worker(new URL("monaco-editor/esm/vs/editor/editor.worker", window.shaper.defaultBaseUrl))
  },
};
loader.config({ monaco });
loader.init();


