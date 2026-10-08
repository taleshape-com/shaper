// SPDX-License-Identifier: MPL-2.0

import { describe, it, expect } from "vitest";
import { toSingleLine, constructCategoryColors } from "./chartUtils";

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
