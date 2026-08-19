// SPDX-License-Identifier: MPL-2.0

import { describe, it, expect } from "vitest";
import { generatePassword } from "./passwordUtils";

describe("passwordUtils", () => {
  describe("generatePassword", () => {
    it("should generate a 16-character password by default", () => {
      const password = generatePassword();
      expect(password).toHaveLength(16);
    });

    it("should generate a password of specified length", () => {
      expect(generatePassword(8)).toHaveLength(8);
      expect(generatePassword(32)).toHaveLength(32);
      expect(generatePassword(64)).toHaveLength(64);
    });

    it("should handle edge case lengths", () => {
      expect(generatePassword(0)).toBe("");
      expect(generatePassword(-5)).toBe("");
      expect(generatePassword(1)).toHaveLength(1);
      expect(generatePassword(2)).toHaveLength(2);
    });

    it("should contain uppercase, lowercase, and digit characters for lengths >= 3", () => {
      for (let i = 0; i < 20; i++) {
        const password = generatePassword(16);
        expect(/[A-Z]/.test(password)).toBe(true);
        expect(/[a-z]/.test(password)).toBe(true);
        expect(/[0-9]/.test(password)).toBe(true);
      }
    });

    it("should only contain valid alphanumeric characters", () => {
      for (let i = 0; i < 20; i++) {
        const password = generatePassword(32);
        expect(/^[A-Za-z0-9]+$/.test(password)).toBe(true);
      }
    });

    it("should produce different passwords on consecutive calls", () => {
      const p1 = generatePassword(16);
      const p2 = generatePassword(16);
      expect(p1).not.toBe(p2);
    });
  });
});
