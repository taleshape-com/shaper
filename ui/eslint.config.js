// SPDX-License-Identifier: MPL-2.0

import js from "@eslint/js";
import globals from "globals";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";

export default tseslint.config(
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    plugins: {
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "react-refresh/only-export-components": [
        "warn",
        { allowConstantExport: true },
      ],
      "@typescript-eslint/no-explicit-any": "off",
      // Formatting rules
      indent: ["error", 2],
      quotes: ["error", "double"],
      semi: ["error", "always"],
      "comma-dangle": ["error", "always-multiline"],
      "object-curly-spacing": ["error", "always"],
      "array-bracket-spacing": ["error", "never"],
      "space-before-function-paren": ["error", "always"],
      "keyword-spacing": ["error", { before: true, after: true }],
      "space-infix-ops": "error",
      "comma-spacing": ["error", { before: false, after: true }],
      "brace-style": ["error", "1tbs", { allowSingleLine: true }],
      curly: ["error", "multi-line"],
      "eol-last": ["error", "always"],
      "no-trailing-spaces": "error",
      "no-multiple-empty-lines": ["error", { max: 1, maxEOF: 0 }],
    },
  });
