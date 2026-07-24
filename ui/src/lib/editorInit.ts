// SPDX-License-Identifier: MPL-2.0

import { loader } from "@monaco-editor/react";
import * as monaco from "monaco-editor";

// Initialize Monaco Editor from source files instead of CDN
self.MonacoEnvironment = {
  getWorker () {
    const defaultBaseUrl = window.shaper.defaultBaseUrl || "/";
    const baseUrl = defaultBaseUrl.startsWith("http://") || defaultBaseUrl.startsWith("https://")
      ? defaultBaseUrl
      : `${window.location.origin}${defaultBaseUrl.startsWith("/") ? "" : "/"}${defaultBaseUrl}`;
    const normalizedBaseUrl = baseUrl.endsWith("/") ? baseUrl : `${baseUrl}/`;
    return new Worker(new URL("monaco-editor/esm/vs/editor/editor.worker", normalizedBaseUrl));
  },
};
loader.config({ monaco });
loader.init();
