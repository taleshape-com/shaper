// SPDX-License-Identifier: MPL-2.0

// Add type definition for the global shaper object
declare global {
  interface Window {
    shaper: {
      defaultBaseUrl: string;
      customCSS?: string;
    };
  }
}
