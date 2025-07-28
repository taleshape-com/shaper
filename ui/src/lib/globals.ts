// Add type definition for the global shaper object
declare global {
  interface Window {
    shaper: {
      defaultBaseUrl: string;
      customCSS?: string;
    };
  }
}


