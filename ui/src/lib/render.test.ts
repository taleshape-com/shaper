// SPDX-License-Identifier: MPL-2.0

import { describe, it, expect } from "vitest";
import { formatValue, scopeDashboardCSS } from "./render";

describe("formatValue with numbers", () => {
  const columnType = "number";
  const shouldFormat = true;

  it("should not format integer with <= 4 digits", () => {
    expect(formatValue(0, columnType, shouldFormat)).toBe("0");
    expect(formatValue(12, columnType, shouldFormat)).toBe("12");
    expect(formatValue(123, columnType, shouldFormat)).toBe("123");
    expect(formatValue(1234, columnType, shouldFormat)).toBe("1234");
    expect(formatValue(-1234, columnType, shouldFormat)).toBe("-1234");
  });

  it("should format integer with >= 5 digits using thin space", () => {
    expect(formatValue(12345, columnType, shouldFormat)).toBe("12 345");
    expect(formatValue(123456, columnType, shouldFormat)).toBe("123 456");
    expect(formatValue(1234567, columnType, shouldFormat)).toBe("1 234 567");
    expect(formatValue(-12345, columnType, shouldFormat)).toBe("-12 345");
  });

  it("should not format non-integer with <= 4 digits on both sides", () => {
    expect(formatValue(1234.5678, columnType, shouldFormat)).toBe("1234.5678");
    expect(formatValue(-1234.5678, columnType, shouldFormat)).toBe("-1234.5678");
  });

  it("should format non-integer with >= 5 digits on left side, but <= 4 on right", () => {
    expect(formatValue(12345.5678, columnType, shouldFormat)).toBe("12 345.5678");
    expect(formatValue(-12345.5678, columnType, shouldFormat)).toBe("-12 345.5678");
  });

  it("should format non-integer with <= 4 digits on left side, but >= 5 on right", () => {
    expect(formatValue(1234.56789, columnType, shouldFormat)).toBe("1234.567 89");
    expect(formatValue(1234.567891, columnType, shouldFormat)).toBe("1234.567 891");
    expect(formatValue(1234.5678912, columnType, shouldFormat)).toBe("1234.567 891 2");
    expect(formatValue(-1234.5678912, columnType, shouldFormat)).toBe("-1234.567 891 2");
  });

  it("should format non-integer with >= 5 digits on both sides", () => {
    expect(formatValue(12345.56789, columnType, shouldFormat)).toBe("12 345.567 89");
    expect(formatValue(123456.5678912, columnType, shouldFormat)).toBe("123 456.567 891 2");
    expect(formatValue(-123456.5678912, columnType, shouldFormat)).toBe("-123 456.567 891 2");
  });

  it("should respect shortFormat by rounding to 2 decimal places before formatting", () => {
    // shortFormat = true should round 12345.6789 to 12345.68, then format to 12 345.68
    expect(formatValue(12345.6789, columnType, shouldFormat, true)).toBe("12 345.68");
    expect(formatValue(1234.5678, columnType, shouldFormat, true)).toBe("1234.57");
    expect(formatValue(12.345, columnType, shouldFormat, true)).toBe("12.35");
  });

  it("should not format if shouldFormatNumbers is false or columnType is not number", () => {
    expect(formatValue(12345, "number", false)).toBe(12345);
    expect(formatValue(12345, "year", true)).toBe("1970"); // parsed as timestamp 12345ms -> 1970
  });

  it("should handle exponents safely by leaving them as-is", () => {
    expect(formatValue(1.234567e+10, columnType, shouldFormat)).toBe("12 345 670 000"); // 11 digits, so it formats to 12 345 670 000
    // If it actually returns 'e' in string representation (e.g. 1e+20)
    expect(formatValue(1e21, columnType, shouldFormat)).toBe("1e+21");
  });
});

describe("formatValue with timestamp", () => {
  it("should include hour and minute when shortFormat is true", () => {
    const formatted = formatValue("2024-01-15T08:30:00Z", "timestamp", true, true);
    expect(formatted).toContain("08:30");
    expect(formatted).not.toContain(":00");
  });

  it("should include seconds when shortFormat is false", () => {
    const formatted = formatValue("2024-01-15T08:30:45Z", "timestamp", true, false);
    expect(formatted).toContain("08:30:45");
  });
});

describe("formatValue with struct, map, object", () => {
  it("should stringify struct object value", () => {
    const val = { name: "Alice", age: 30 };
    expect(formatValue(val, "struct")).toBe(JSON.stringify(val));
    expect(formatValue(val, "map")).toBe(JSON.stringify(val));
    expect(formatValue(val, "object")).toBe(JSON.stringify(val));
  });
});

describe("scopeDashboardCSS", () => {
  it("should return empty string for empty input", () => {
    expect(scopeDashboardCSS("", "shaper-instance-123")).toBe("");
  });

  it("should scope rules to the specific instance class", () => {
    const css = ".card { padding: 16px; }\nh1 { color: red; }";
    const scoped = scopeDashboardCSS(css, "shaper-instance-123");
    expect(scoped).toContain("@scope (.shaper-instance-123)");
    expect(scoped).toContain(".shaper-instance-123 {");
    expect(scoped).toContain(".card { padding: 16px; }");
    expect(scoped).toContain("h1 { color: red; }");
  });

  it("should hoist @import rules above @scope block", () => {
    const css = "@import url(\"https://fonts.googleapis.com/css?family=Inter\");\n.card { border-radius: 4px; }";
    const scoped = scopeDashboardCSS(css, "shaper-instance-abc");
    expect(scoped.startsWith("@import url(\"https://fonts.googleapis.com/css?family=Inter\");\n@supports")).toBe(true);
    expect(scoped).toContain("@scope (.shaper-instance-abc)");
  });

  it("should map body, html, :root to :scope in @scope block and & in nesting block", () => {
    const css = "body { background: #121212; }\nhtml { font-size: 16px; }\n:root { --accent: blue; }";
    const scoped = scopeDashboardCSS(css, "shaper-instance-xyz");
    // Under @scope block
    expect(scoped).toContain(":scope { background: #121212; }");
    expect(scoped).toContain(":scope { font-size: 16px; }");
    expect(scoped).toContain(":scope { --accent: blue; }");
    // Under nesting fallback block
    expect(scoped).toContain("& { background: #121212; }");
    expect(scoped).toContain("& { font-size: 16px; }");
    expect(scoped).toContain("& { --accent: blue; }");
  });
});
