// SPDX-License-Identifier: MPL-2.0

import { describe, it, expect } from "vitest";
import { addYears, isSameMonth, formatDate } from "./dateUtils";

describe("dateUtils helpers", () => {
  describe("addYears", () => {
    it("should add positive years correctly", () => {
      const date = new Date(2020, 5, 15); // June 15, 2020
      const result = addYears(date, 5);
      expect(result.getFullYear()).toBe(2025);
      expect(result.getMonth()).toBe(5);
      expect(result.getDate()).toBe(15);
    });

    it("should subtract negative years correctly", () => {
      const date = new Date(2020, 5, 15); // June 15, 2020
      const result = addYears(date, -5);
      expect(result.getFullYear()).toBe(2015);
      expect(result.getMonth()).toBe(5);
      expect(result.getDate()).toBe(15);
    });

    it("should return a new Date instance and not modify original", () => {
      const date = new Date(2020, 5, 15);
      const result = addYears(date, 1);
      expect(result).not.toBe(date);
      expect(date.getFullYear()).toBe(2020);
    });
  });

  describe("isSameMonth", () => {
    it("should return true for same month and same year", () => {
      const d1 = new Date(2020, 5, 15);
      const d2 = new Date(2020, 5, 20);
      expect(isSameMonth(d1, d2)).toBe(true);
    });

    it("should return false for different months in same year", () => {
      const d1 = new Date(2020, 5, 15);
      const d2 = new Date(2020, 6, 15);
      expect(isSameMonth(d1, d2)).toBe(false);
    });

    it("should return false for same month in different years", () => {
      const d1 = new Date(2020, 5, 15);
      const d2 = new Date(2021, 5, 15);
      expect(isSameMonth(d1, d2)).toBe(false);
    });
  });

  describe("formatDate", () => {
    it("should format 'MMM dd, yyyy' correctly", () => {
      const date = new Date(2026, 5, 15); // June 15, 2026
      expect(formatDate(date, "MMM dd, yyyy")).toBe("Jun 15, 2026");
    });

    it("should format 'MMM, yyyy' correctly", () => {
      const date = new Date(2026, 5, 15); // June 15, 2026
      expect(formatDate(date, "MMM, yyyy")).toBe("Jun, 2026");
    });

    it("should fallback/replace general tokens", () => {
      const date = new Date(2026, 5, 15); // June 15, 2026
      expect(formatDate(date, "yyyy-MMM-dd")).toBe("2026-Jun-15");
    });
  });
});
