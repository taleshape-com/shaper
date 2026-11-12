// SPDX-License-Identifier: MPL-2.0

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import * as echarts from "echarts";
import {
  toSingleLine,
  constructCategoryColors,
  getComputedCssValue,
  getChartFont,
  getDisplayFont,
  getThemeColors,
  getNiceNumber,
  getNiceAxisRange,
  extractMarkLineNumber,
  getMarkLineAxisBounds,
  getMarkLineLabelAlignment,
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

describe("getNiceNumber and getNiceAxisRange", () => {
  it("computes nice numbers correctly", () => {
    expect(getNiceNumber(42, true)).toBe(50);
    expect(getNiceNumber(12, true)).toBe(10);
    expect(getNiceNumber(0, true)).toBe(1);
    expect(getNiceNumber(-10, true)).toBe(1);
  });

  it("computes nice axis range correctly", () => {
    expect(getNiceAxisRange(0, 42)).toEqual({ niceMin: 0, niceMax: 50 });
    expect(getNiceAxisRange(20, 150)).toEqual({ niceMin: 0, niceMax: 150 });
    expect(getNiceAxisRange(-15, 30)).toEqual({ niceMin: -20, niceMax: 30 });
    expect(getNiceAxisRange(0, 0)).toEqual({ niceMin: 0, niceMax: 1 });
    expect(getNiceAxisRange(50, 50)).toEqual({ niceMin: 0, niceMax: 60 });
    expect(getNiceAxisRange(-50, -50)).toEqual({ niceMin: -60, niceMax: 0 });
    expect(getNiceAxisRange(0.02, 0.08)).toEqual({ niceMin: 0.02, niceMax: 0.08 });
  });
});

describe("extractMarkLineNumber", () => {
  it("extracts numeric values", () => {
    expect(extractMarkLineNumber(50)).toBe(50);
    expect(extractMarkLineNumber("50")).toBe(50);
    expect(extractMarkLineNumber(0)).toBe(0);
    expect(extractMarkLineNumber(-15.5)).toBe(-15.5);
    expect(extractMarkLineNumber(NaN)).toBeUndefined();
    expect(extractMarkLineNumber(Infinity)).toBeUndefined();
    expect(extractMarkLineNumber("abc")).toBeUndefined();
  });

  it("extracts time values when isTime is true", () => {
    const ts = 1704153600000;
    expect(extractMarkLineNumber(ts, true)).toBe(ts);
    expect(extractMarkLineNumber("2024-01-02T00:00:00.000Z", true)).toBe(ts);
  });
});

describe("getMarkLineAxisBounds", () => {
  it("returns undefined callbacks if no valid values", () => {
    const bounds = getMarkLineAxisBounds([]);
    expect(bounds.min).toBeUndefined();
    expect(bounds.max).toBeUndefined();
  });

  it("returns undefined when markLine is inside data range", () => {
    const bounds = getMarkLineAxisBounds([25], { scale: false });
    expect(bounds.min!({ min: 20, max: 30 })).toBeUndefined();
    expect(bounds.max!({ min: 20, max: 30 })).toBeUndefined();
  });

  it("expands max when markLine is above data max", () => {
    const bounds = getMarkLineAxisBounds([42], { scale: false });
    expect(bounds.min!({ min: 20, max: 30 })).toBeUndefined();
    expect(bounds.max!({ min: 20, max: 30 })).toBe(50);
  });

  it("expands min when markLine is below 0 or data min", () => {
    const bounds = getMarkLineAxisBounds([-15], { scale: false });
    expect(bounds.min!({ min: 20, max: 30 })).toBe(-20);
    expect(bounds.max!({ min: 20, max: 30 })).toBeUndefined();
  });

  it("handles empty data properly", () => {
    const bounds = getMarkLineAxisBounds([50], { scale: false });
    expect(bounds.min!({ min: NaN, max: NaN })).toBe(0);
    expect(bounds.max!({ min: NaN, max: NaN })).toBe(50);

    const negBounds = getMarkLineAxisBounds([-15], { scale: false });
    expect(negBounds.min!({ min: NaN, max: NaN })).toBe(-15);
    expect(negBounds.max!({ min: NaN, max: NaN })).toBe(0);
  });

  it("handles time axis bounds with empty and non-empty data", () => {
    const t = 1704153600000;
    const timeBounds = getMarkLineAxisBounds([t], { isTime: true, scale: true });
    expect(timeBounds.min!({ min: NaN, max: NaN })).toBe(t - 86400000);
    expect(timeBounds.max!({ min: NaN, max: NaN })).toBe(t + 86400000);

    const tEarly = t - 2000000;
    const tLate = t + 2000000;
    const earlyBounds = getMarkLineAxisBounds([tEarly], { isTime: true, scale: true });
    expect(earlyBounds.min!({ min: t, max: t + 1000 })).toBe(tEarly);
    expect(earlyBounds.max!({ min: t, max: t + 1000 })).toBeUndefined();

    const lateBounds = getMarkLineAxisBounds([tLate], { isTime: true, scale: true });
    expect(lateBounds.min!({ min: t, max: t + 1000 })).toBeUndefined();
    expect(lateBounds.max!({ min: t, max: t + 1000 })).toBe(tLate);
  });

  it("handles multiple markLines with both min and max outside range", () => {
    const bounds = getMarkLineAxisBounds([-50, 150], { scale: false });
    expect(bounds.min!({ min: 10, max: 20 })).toBe(-50);
    expect(bounds.max!({ min: 10, max: 20 })).toBe(150);
  });

  it("handles decimal markLine values", () => {
    const bounds = getMarkLineAxisBounds([0.08], { scale: false });
    expect(bounds.min!({ min: 0.01, max: 0.04 })).toBeUndefined();
    expect(bounds.max!({ min: 0.01, max: 0.04 })).toBe(0.08);
  });
});

describe("ECharts markLine visibility with getMarkLineAxisBounds", () => {
  it("renders yLine outside data range in SVG", () => {
    const yBounds = getMarkLineAxisBounds([50], { scale: false });
    const chart = echarts.init(null, null, { renderer: "svg", ssr: true, width: 500, height: 300 });
    chart.setOption({
      xAxis: { type: "category", data: ["A", "B"] },
      yAxis: {
        type: "value",
        min: yBounds.min,
        max: yBounds.max,
      },
      series: [
        { type: "line", data: [10, 20] },
        {
          type: "line",
          markLine: {
            data: [{ yAxis: 50, label: { formatter: "Goal50" } }],
          },
        },
      ],
    });
    const svg = chart.renderToSVGString();
    expect(svg).toContain("Goal50");
  });

  it("renders negative yLine in SVG", () => {
    const yBounds = getMarkLineAxisBounds([-25], { scale: false });
    const chart = echarts.init(null, null, { renderer: "svg", ssr: true, width: 500, height: 300 });
    chart.setOption({
      xAxis: { type: "category", data: ["A", "B"] },
      yAxis: {
        type: "value",
        min: yBounds.min,
        max: yBounds.max,
      },
      series: [
        { type: "line", data: [10, 20] },
        {
          type: "line",
          markLine: {
            data: [{ yAxis: -25, label: { formatter: "Loss25" } }],
          },
        },
      ],
    });
    const svg = chart.renderToSVGString();
    expect(svg).toContain("Loss25");
  });

  it("renders xLine on continuous value xAxis outside data range", () => {
    const xBounds = getMarkLineAxisBounds([60], { scale: true });
    const chart = echarts.init(null, null, { renderer: "svg", ssr: true, width: 500, height: 300 });
    chart.setOption({
      xAxis: {
        type: "value",
        min: xBounds.min,
        max: xBounds.max,
      },
      yAxis: { type: "value" },
      series: [
        { type: "line", data: [[10, 10], [20, 20]] },
        {
          type: "line",
          markLine: {
            data: [{ xAxis: 60, label: { formatter: "Milestone60" } }],
          },
        },
      ],
    });
    const svg = chart.renderToSVGString();
    expect(svg).toContain("Milestone60");
  });

  it("renders markLine on empty chart without data", () => {
    const yBounds = getMarkLineAxisBounds([30], { scale: false });
    const chart = echarts.init(null, null, { renderer: "svg", ssr: true, width: 500, height: 300 });
    chart.setOption({
      xAxis: { type: "category", data: [""] },
      yAxis: {
        type: "value",
        min: yBounds.min,
        max: yBounds.max,
      },
      series: [
        {
          type: "line",
          data: [],
          markLine: {
            data: [{ yAxis: 30, label: { formatter: "Target30" } }],
          },
        },
      ],
    });
    const svg = chart.renderToSVGString();
    expect(svg).toContain("Target30");
  });

  it("renders Scatterplot yLine outside data range in SVG", () => {
    const yBounds = getMarkLineAxisBounds([85], { scale: false });
    const chart = echarts.init(null, null, { renderer: "svg", ssr: true, width: 500, height: 300 });
    chart.setOption({
      xAxis: { type: "category", data: ["A", "B"] },
      yAxis: {
        type: "value",
        min: yBounds.min,
        max: yBounds.max,
      },
      series: [
        { type: "scatter", data: [15, 25] },
        {
          type: "scatter",
          markLine: {
            data: [{ yAxis: 85, label: { formatter: "ScatterY85" } }],
          },
        },
      ],
    });
    const svg = chart.renderToSVGString();
    expect(svg).toContain("ScatterY85");
  });

  it("renders Scatterplot xLine outside continuous data range in SVG", () => {
    const xBounds = getMarkLineAxisBounds([100], { scale: true });
    const chart = echarts.init(null, null, { renderer: "svg", ssr: true, width: 500, height: 300 });
    chart.setOption({
      xAxis: {
        type: "value",
        min: xBounds.min,
        max: xBounds.max,
      },
      yAxis: { type: "value" },
      series: [
        { type: "scatter", data: [[10, 20], [20, 30]] },
        {
          type: "scatter",
          markLine: {
            data: [{ xAxis: 100, label: { formatter: "ScatterX100" } }],
          },
        },
      ],
    });
    const svg = chart.renderToSVGString();
    expect(svg).toContain("ScatterX100");
  });

  it("renders Boxplot yLine outside data range in SVG", () => {
    const yBounds = getMarkLineAxisBounds([90], { scale: false });
    const chart = echarts.init(null, null, { renderer: "svg", ssr: true, width: 500, height: 300 });
    chart.setOption({
      xAxis: { type: "category", data: ["Group A"] },
      yAxis: {
        type: "value",
        min: yBounds.min,
        max: yBounds.max,
      },
      series: [
        {
          type: "boxplot",
          data: [[10, 20, 30, 40, 50]],
        },
        {
          type: "line",
          markLine: {
            data: [{ yAxis: 90, label: { formatter: "BoxGoal90" } }],
          },
        },
      ],
    });
    const svg = chart.renderToSVGString();
    expect(svg).toContain("BoxGoal90");
  });

  it("renders Boxplot xLine on missing category in SVG", () => {
    const chart = echarts.init(null, null, { renderer: "svg", ssr: true, width: 500, height: 300 });
    chart.setOption({
      xAxis: { type: "category", data: ["Group A", "Group B"] },
      yAxis: { type: "value" },
      series: [
        {
          type: "boxplot",
          data: [[10, 20, 30, 40, 50]], // only Group A has data
        },
        {
          type: "line",
          markLine: {
            data: [{ xAxis: "Group B", label: { formatter: "EventB" } }],
          },
        },
      ],
    });
    const svg = chart.renderToSVGString();
    expect(svg).toContain("EventB");
  });

  it("renders empty Boxplot with yLine in SVG", () => {
    const yBounds = getMarkLineAxisBounds([50], { scale: false });
    const chart = echarts.init(null, null, { renderer: "svg", ssr: true, width: 500, height: 300 });
    chart.setOption({
      xAxis: { type: "category", data: [""] },
      yAxis: {
        type: "value",
        min: yBounds.min,
        max: yBounds.max,
      },
      series: [
        {
          type: "line",
          markLine: {
            data: [{ yAxis: 50, label: { formatter: "BoxEmpty50" } }],
          },
        },
      ],
    });
    const svg = chart.renderToSVGString();
    expect(svg).toContain("BoxEmpty50");
  });
});

describe("getMarkLineLabelAlignment", () => {
  it("returns default center when options are not provided", () => {
    expect(getMarkLineLabelAlignment("test")).toEqual({ align: "center" });
    expect(getMarkLineLabelAlignment(50)).toEqual({ align: "center" });
  });

  describe("categorical axis", () => {
    const categories = ["Jan", "Feb", "Mar", "Apr", "May"];

    it("aligns right with inward padding for the last category", () => {
      expect(getMarkLineLabelAlignment("May", { categories })).toEqual({
        align: "right",
        padding: [0, 4, 0, 0],
      });
    });

    it("aligns left with inward padding for the first category", () => {
      expect(getMarkLineLabelAlignment("Jan", { categories })).toEqual({
        align: "left",
        padding: [0, 0, 0, 4],
      });
    });

    it("aligns center for middle categories", () => {
      expect(getMarkLineLabelAlignment("Feb", { categories })).toEqual({
        align: "center",
      });
      expect(getMarkLineLabelAlignment("Mar", { categories })).toEqual({
        align: "center",
      });
      expect(getMarkLineLabelAlignment("Apr", { categories })).toEqual({
        align: "center",
      });
    });

    it("matches dates even if represented in different format/string", () => {
      const dateCategories = ["2024-01-01T00:00:00.000Z", "2024-02-01T00:00:00.000Z", "2024-03-01T00:00:00.000Z"];
      expect(
        getMarkLineLabelAlignment("2024-03-01T00:00:00Z", { categories: dateCategories }),
      ).toEqual({
        align: "right",
        padding: [0, 4, 0, 0],
      });
      expect(
        getMarkLineLabelAlignment(new Date("2024-01-01T00:00:00.000Z").getTime(), { categories: dateCategories }),
      ).toEqual({
        align: "left",
        padding: [0, 0, 0, 4],
      });
    });

    it("falls back to center when categories array has 1 or 0 items", () => {
      expect(getMarkLineLabelAlignment("Single", { categories: ["Single"] })).toEqual({
        align: "center",
      });
      expect(getMarkLineLabelAlignment("None", { categories: [] })).toEqual({
        align: "center",
      });
    });
  });

  describe("continuous axis", () => {
    it("aligns right when value is near or at max (>= 90% of range)", () => {
      expect(
        getMarkLineLabelAlignment(95, { isContinuous: true, min: 0, max: 100 }),
      ).toEqual({
        align: "right",
        padding: [0, 4, 0, 0],
      });
      expect(
        getMarkLineLabelAlignment(90, { isContinuous: true, min: 0, max: 100 }),
      ).toEqual({
        align: "right",
        padding: [0, 4, 0, 0],
      });
      expect(
        getMarkLineLabelAlignment(100, { isContinuous: true, min: 0, max: 100 }),
      ).toEqual({
        align: "right",
        padding: [0, 4, 0, 0],
      });
    });

    it("aligns left when value is near or at min (<= 10% of range)", () => {
      expect(
        getMarkLineLabelAlignment(5, { isContinuous: true, min: 0, max: 100 }),
      ).toEqual({
        align: "left",
        padding: [0, 0, 0, 4],
      });
      expect(
        getMarkLineLabelAlignment(10, { isContinuous: true, min: 0, max: 100 }),
      ).toEqual({
        align: "left",
        padding: [0, 0, 0, 4],
      });
      expect(
        getMarkLineLabelAlignment(0, { isContinuous: true, min: 0, max: 100 }),
      ).toEqual({
        align: "left",
        padding: [0, 0, 0, 4],
      });
    });

    it("aligns center when value is in the middle 80% range", () => {
      expect(
        getMarkLineLabelAlignment(50, { isContinuous: true, min: 0, max: 100 }),
      ).toEqual({
        align: "center",
      });
      expect(
        getMarkLineLabelAlignment(15, { isContinuous: true, min: 0, max: 100 }),
      ).toEqual({
        align: "center",
      });
      expect(
        getMarkLineLabelAlignment(85, { isContinuous: true, min: 0, max: 100 }),
      ).toEqual({
        align: "center",
      });
    });

    it("renders ECharts SVG with right-aligned markLine label tucked inward", () => {
      const alignment = getMarkLineLabelAlignment("D", { categories: ["A", "B", "C", "D"] });
      const chart = echarts.init(null, null, { renderer: "svg", ssr: true, width: 500, height: 300 });
      chart.setOption({
        grid: { left: 50, right: 20, top: 40, bottom: 30 },
        xAxis: { type: "category", data: ["A", "B", "C", "D"] },
        yAxis: { type: "value" },
        series: [{
          type: "line",
          data: [10, 20, 30, 40],
          markLine: {
            data: [{
              xAxis: "D",
              label: {
                formatter: "RightEdgeLabel",
                position: "end",
                align: alignment.align,
                padding: alignment.padding,
              },
            }],
          },
        }],
      });
      const svg = chart.renderToSVGString();
      expect(svg).toContain("RightEdgeLabel");
      expect(svg).toContain("text-anchor=\"end\"");
    });

    it("renders horizontal bar chart with vertical markLine label horizontally at the top", () => {
      const chart = echarts.init(null, null, { renderer: "svg", ssr: true, width: 500, height: 300 });
      chart.setOption({
        grid: { left: 80, right: 20, top: 40, bottom: 40 },
        xAxis: { type: "value", min: 0, max: 100 },
        yAxis: { type: "category", data: ["Ingestion", "Transformation", "Indexing"] },
        series: [{
          type: "bar",
          data: [22, 48, 31],
          markLine: {
            data: [{
              xAxis: 80,
              label: {
                formatter: "Max SLA (80s)",
                position: "end",
                distance: 3.8,
                align: "center",
              },
            }],
          },
        }],
      });
      const svg = chart.renderToSVGString();
      expect(svg).toContain("Max SLA (80s)");
      // Verify it is placed horizontally using translate() without 90-degree matrix rotation
      const match = svg.match(/<text[^>]*>Max SLA \(80s\)<\/text>/);
      expect(match).toBeTruthy();
      expect(match![0]).toContain("translate");
      expect(match![0]).not.toContain("matrix(0,-1,1,0");
    });
  });
});
