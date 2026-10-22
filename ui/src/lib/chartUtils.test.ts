// SPDX-License-Identifier: MPL-2.0

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import {
  toSingleLine,
  constructCategoryColors,
  getComputedCssValue,
  getChartFont,
  getDisplayFont,
  getThemeColors,
} from "./chartUtils";

describe("toSingleLine", () => {
  it("should return single-line string unchanged", () => {
    expect(toSingleLine("hello world")).toBe("hello world");
    expect(toSingleLine("")).toBe("");
  });

  it("should replace newline with space", () => {
    expect(toSingleLine("hello\nworld")).toBe("hello world");
    expect(toSingleLine("axis\nlabel\nhere")).toBe("axis label here");
  });

  it("should replace windows CRLF with space", () => {
    expect(toSingleLine("hello\r\nworld")).toBe("hello world");
    expect(toSingleLine("foo\r\nbar\r\nbaz")).toBe("foo bar baz");
  });

  it("should replace classic Mac CR with space", () => {
    expect(toSingleLine("hello\rworld")).toBe("hello world");
  });

  it("should handle consecutive newlines", () => {
    expect(toSingleLine("hello\n\nworld")).toBe("hello  world");
    expect(toSingleLine("hello\r\n\r\nworld")).toBe("hello  world");
  });

  it("should handle leading and trailing newlines", () => {
    expect(toSingleLine("\nhello\n")).toBe(" hello ");
    expect(toSingleLine("\r\nworld\r\n")).toBe(" world ");
  });

  it("should preserve non-string values", () => {
    expect(toSingleLine(null)).toBeNull();
    expect(toSingleLine(undefined)).toBeUndefined();
    expect(toSingleLine(123)).toBe(123);
    expect(toSingleLine(0)).toBe(0);
  });
});

describe("constructCategoryColors", () => {
  it("should assign colors and populate both cleaned and original category keys", () => {
    const categories = ["North\nAmerica", "South\r\nAmerica", "Europe"];
    const colorsByCategory = {
      "North America": "#ff0000",
      "South\r\nAmerica": "#00ff00",
    };

    const categoryColors = constructCategoryColors(categories, colorsByCategory, false);

    expect(categoryColors.get("North America")).toBe("#ff0000");
    expect(categoryColors.get("North\nAmerica")).toBe("#ff0000");

    expect(categoryColors.get("South America")).toBe("#00ff00");
    expect(categoryColors.get("South\r\nAmerica")).toBe("#00ff00");

    expect(categoryColors.has("Europe")).toBe(true);
  });
});

describe("getComputedCssValue and theme utilities", () => {
  it("should return fallback var when document is undefined", () => {
    expect(getComputedCssValue("--shaper-primary-color")).toBe("var(--shaper-primary-color)");
    expect(getChartFont()).toBe("sans-serif");
  });

  describe("with mock DOM", () => {
    let originalDoc: any;
    let originalWin: any;
    let originalGetComputedStyle: any;

    beforeEach(() => {
      originalDoc = (global as any).document;
      originalWin = (global as any).window;
      originalGetComputedStyle = (global as any).getComputedStyle;
    });

    afterEach(() => {
      (global as any).document = originalDoc;
      (global as any).window = originalWin;
      (global as any).getComputedStyle = originalGetComputedStyle;
    });

    it("should resolve scoped css values from target element or chart id", () => {
      const scopedStyle = {
        getPropertyValue: (prop: string) => {
          if (prop === "--shaper-font") return "CustomFont, sans-serif";
          if (prop === "--shaper-primary-color-alternate") return "#ff00ff";
          return "";
        },
      };

      const rootStyle = {
        getPropertyValue: (prop: string) => {
          if (prop === "--shaper-font") return "RootFont";
          if (prop === "--shaper-background-color") return "#ffffff";
          return "";
        },
      };

      const mockScopeEl = {
        closest: () => mockScopeEl,
      };

      const mockChartEl = {
        closest: (sel: string) => (sel === ".shaper-scope" ? mockScopeEl : null),
      };

      const mockDoc = {
        documentElement: {},
        querySelector: (sel: string) => {
          if (sel === "[data-chart-id=\"test-chart\"]") return mockChartEl;
          if (sel === ".shaper-scope") return mockScopeEl;
          return null;
        },
      };

      (global as any).document = mockDoc;
      (global as any).window = {};
      (global as any).getComputedStyle = (el: any) => {
        if (el === mockScopeEl) return scopedStyle;
        return rootStyle;
      };

      // 1. Target chartId resolves closest .shaper-scope
      expect(getComputedCssValue("--shaper-font", "test-chart")).toBe("CustomFont, sans-serif");
      expect(getChartFont("test-chart")).toBe("CustomFont, sans-serif");
      expect(getDisplayFont("test-chart")).toBe("sans-serif"); // fallback when empty
      expect(getThemeColors(false, "test-chart").primaryColor).toBe("#ff00ff");

      // 2. Falls back to documentElement when variable is not defined on scoped element
      expect(getComputedCssValue("--shaper-background-color", "test-chart")).toBe("#ffffff");

      // 3. Falls back to var() when not defined anywhere
      expect(getComputedCssValue("--undefined-var", "test-chart")).toBe("var(--undefined-var)");

      // 4. Works with direct element passed
      expect(getComputedCssValue("--shaper-font", mockChartEl as any)).toBe("CustomFont, sans-serif");
    });

    it("should handle numeric chart IDs like '3-1' safely even when querySelector throws", () => {
      const scopedStyle = {
        getPropertyValue: (prop: string) => {
          if (prop === "--shaper-font") return "ScopedChartFont";
          return "";
        },
      };

      const mockScopeEl = {
        closest: () => mockScopeEl,
      };

      const mockChartEl = {
        closest: (sel: string) => (sel === ".shaper-scope" ? mockScopeEl : null),
      };

      const mockDoc = {
        documentElement: {},
        querySelector: (sel: string) => {
          if (sel === "[data-chart-id=\"3-1\"]") return mockChartEl;
          if (sel === ".shaper-scope") return mockScopeEl;
          if (sel === "3-1") {
            throw new Error("Failed to execute 'querySelector' on 'Document': '3-1' is not a valid selector.");
          }
          return null;
        },
        getElementById: () => null,
      };

      (global as any).document = mockDoc;
      (global as any).window = {};
      (global as any).getComputedStyle = (el: any) => {
        if (el === mockScopeEl) return scopedStyle;
        return { getPropertyValue: () => "" };
      };

      // 1. Mounted chart id '3-1' resolves properly
      expect(getComputedCssValue("--shaper-font", "3-1")).toBe("ScopedChartFont");

      // 2. Unmounted chart id '3-2' where querySelector throws on fallback '3-2' does not crash
      mockDoc.querySelector = (sel: string) => {
        if (sel === ".shaper-scope") return mockScopeEl;
        if (sel === "3-2") {
          throw new Error("Failed to execute 'querySelector' on 'Document': '3-2' is not a valid selector.");
        }
        return null;
      };

      expect(() => getComputedCssValue("--shaper-font", "3-2")).not.toThrow();
      expect(getComputedCssValue("--shaper-font", "3-2")).toBe("ScopedChartFont");
    });
  });
});
