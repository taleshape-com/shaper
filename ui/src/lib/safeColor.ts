// SPDX-License-Identifier: MPL-2.0

// HTML-escaping (echarts.format.encodeHTML) does not block CSS injection
// when a value is interpolated into a `style="..."` attribute, because it
// leaves `;`, `:`, `(`, `)` untouched. SQL-supplied colors (the `color`-tagged
// column on pie/bar/line/boxplot dashboards) reach the tooltip style
// attribute, so we whitelist known-safe CSS color syntaxes and drop the
// rest. Anything unrecognised becomes an empty string and the chart falls
// back to its theme color.

const HEX = /^#[0-9a-fA-F]{3,8}$/;
const NAMED = /^[a-zA-Z]+$/;
const COLOR_FN = /^(rgb|rgba|hsl|hsla|hwb|lab|lch|oklab|oklch|color)\([0-9a-zA-Z\s,./%-]+\)$/;
const VAR_REF = /^var\(--[a-zA-Z0-9_-]+\)$/;

export function safeColor (input: string | null | undefined): string {
  if (input == null) return "";
  const trimmed = String(input).trim();
  if (!trimmed) return "";
  if (HEX.test(trimmed)) return trimmed;
  if (NAMED.test(trimmed)) return trimmed;
  if (COLOR_FN.test(trimmed)) return trimmed;
  if (VAR_REF.test(trimmed)) return trimmed;
  return "";
}
