// SPDX-License-Identifier: MPL-2.0

import { describe, it, expect } from "vitest";
import {
  isSafeUrl,
  isRelativeOrQueryUrl,
  parseLinkAttributes,
  trimBareUrl,
  splitTextAndUrls,
} from "./linkUtils";

describe("isSafeUrl", () => {
  it("allows safe https, http, mailto, and relative paths", () => {
    expect(isSafeUrl("https://example.com")).toBe(true);
    expect(isSafeUrl("https://example.com/path?foo=bar#hash")).toBe(true);
    expect(isSafeUrl("http://localhost:3000")).toBe(true);
    expect(isSafeUrl("mailto:support@taleshape.com")).toBe(true);
    expect(isSafeUrl("mailto:support@taleshape.com?subject=Help")).toBe(true);
    expect(isSafeUrl("/dashboards/123")).toBe(true);
    expect(isSafeUrl("/view/sales?range=30d")).toBe(true);
    expect(isSafeUrl("./relative/subpath")).toBe(true);
    expect(isSafeUrl("../parent/view")).toBe(true);
    expect(isSafeUrl("#section-kpi")).toBe(true);
    expect(isSafeUrl("?tab=overview&range=30d")).toBe(true);
  });

  it("rejects unsafe schemes and dangerous targets", () => {
    expect(isSafeUrl("javascript:alert(1)")).toBe(false);
    expect(isSafeUrl("JAVASCRIPT:alert(1)")).toBe(false);
    expect(isSafeUrl("javascript: alert(1)")).toBe(false);
    expect(isSafeUrl("data:text/html,<script>alert(1)</script>")).toBe(false);
    expect(isSafeUrl("vbscript:msgbox(1)")).toBe(false);
    expect(isSafeUrl("file:///etc/passwd")).toBe(false);
    expect(isSafeUrl("blob:https://example.com/uuid")).toBe(false);
    expect(isSafeUrl("//evil.com")).toBe(false);
    expect(isSafeUrl("///evil.com")).toBe(false);
    expect(isSafeUrl("")).toBe(false);
    expect(isSafeUrl("http://")).toBe(false);
    expect(isSafeUrl("https://")).toBe(false);
    expect(isSafeUrl("https:// example.com")).toBe(false);
    expect(isSafeUrl("javascript:void(0)")).toBe(false);
  });
});

describe("isRelativeOrQueryUrl", () => {
  it("identifies relative paths and query strings", () => {
    expect(isRelativeOrQueryUrl("/dashboards/123")).toBe(true);
    expect(isRelativeOrQueryUrl("./subpath")).toBe(true);
    expect(isRelativeOrQueryUrl("../parent/view")).toBe(true);
    expect(isRelativeOrQueryUrl("?country=US")).toBe(true);
    expect(isRelativeOrQueryUrl("   ?tab=1   ")).toBe(true);
  });

  it("returns false for fragment hashes, absolute URLs, and empty strings", () => {
    expect(isRelativeOrQueryUrl("#kpi")).toBe(false);
    expect(isRelativeOrQueryUrl("https://example.com")).toBe(false);
    expect(isRelativeOrQueryUrl("http://localhost:3000")).toBe(false);
    expect(isRelativeOrQueryUrl("mailto:support@taleshape.com")).toBe(false);
    expect(isRelativeOrQueryUrl("")).toBe(false);
  });
});

describe("parseLinkAttributes", () => {
  it("leaves target and rel undefined when no attrs provided", () => {
    expect(parseLinkAttributes(undefined)).toEqual({});
    expect(parseLinkAttributes("")).toEqual({});
  });

  it("parses valid target and rel attributes", () => {
    expect(parseLinkAttributes("target=\"_blank\"")).toEqual({
      target: "_blank",
      rel: "noopener noreferrer",
    });
    expect(parseLinkAttributes("target=\"_self\"")).toEqual({
      target: "_self",
      rel: undefined,
    });
    expect(parseLinkAttributes("target=\"_self\" rel=\"nofollow\"")).toEqual({
      target: "_self",
      rel: "nofollow",
    });
    expect(parseLinkAttributes("rel=\"noopener\" target=\"_top\"")).toEqual({
      target: "_top",
      rel: "noopener",
    });
    expect(parseLinkAttributes("target='_parent'")).toEqual({
      target: "_parent",
      rel: undefined,
    });
  });

  it("ignores any attribute other than target and rel", () => {
    const malicious = "target=\"_blank\" onclick=\"alert(1)\" class=\"dangerous\" id=\"pwn\" style=\"display:none\"";
    expect(parseLinkAttributes(malicious)).toEqual({
      target: "_blank",
      rel: "noopener noreferrer",
    });
  });

  it("rejects invalid or unsafe target and rel values", () => {
    expect(parseLinkAttributes("target=\"javascript:alert(1)\"")).toEqual({
      target: undefined,
      rel: undefined,
    });
    expect(parseLinkAttributes("rel=\"<script>alert(1)</script>\" target=\"_self\"")).toEqual({
      target: "_self",
      rel: undefined,
    });
  });
});

describe("trimBareUrl", () => {
  it("trims trailing punctuation from URLs", () => {
    expect(trimBareUrl("https://example.com.")).toEqual({
      url: "https://example.com",
      trailing: ".",
    });
    expect(trimBareUrl("https://example.com,")).toEqual({
      url: "https://example.com",
      trailing: ",",
    });
    expect(trimBareUrl("https://example.com/api?v=1:;!?")).toEqual({
      url: "https://example.com/api?v=1",
      trailing: ":;!?",
    });
  });

  it("handles balanced and unbalanced parens and brackets", () => {
    expect(trimBareUrl("https://example.com/path)")).toEqual({
      url: "https://example.com/path",
      trailing: ")",
    });
    expect(trimBareUrl("https://example.com/path).")).toEqual({
      url: "https://example.com/path",
      trailing: ").",
    });
    expect(trimBareUrl("https://en.wikipedia.org/wiki/Go_(programming_language)")).toEqual({
      url: "https://en.wikipedia.org/wiki/Go_(programming_language)",
      trailing: "",
    });
    expect(trimBareUrl("https://en.wikipedia.org/wiki/Go_(programming_language))")).toEqual({
      url: "https://en.wikipedia.org/wiki/Go_(programming_language)",
      trailing: ")",
    });
    expect(trimBareUrl("https://example.com/page]")).toEqual({
      url: "https://example.com/page",
      trailing: "]",
    });
  });
});

describe("splitTextAndUrls", () => {
  it("handles plain text with no links", () => {
    expect(splitTextAndUrls("Just plain subtitle text")).toEqual([
      { type: "text", content: "Just plain subtitle text" },
    ]);
  });

  it("extracts bare URLs cleanly with surrounding punctuation and displays URL as is", () => {
    const result = splitTextAndUrls("Check documentation at https://example.com/docs.");
    expect(result).toEqual([
      { type: "text", content: "Check documentation at " },
      {
        type: "url",
        href: "https://example.com/docs",
        content: "https://example.com/docs",
        target: "_blank",
        rel: "noopener noreferrer",
      },
      { type: "text", content: "." },
    ]);
  });

  it("extracts markdown links with custom labels", () => {
    const result = splitTextAndUrls("See [our docs](https://example.com/docs) for details!");
    expect(result).toEqual([
      { type: "text", content: "See " },
      {
        type: "url",
        href: "https://example.com/docs",
        content: "our docs",
        target: undefined,
        rel: undefined,
      },
      { type: "text", content: " for details!" },
    ]);
  });

  it("displays URL as is when markdown link label is empty", () => {
    const result = splitTextAndUrls("[](https://example.com/help)");
    expect(result).toEqual([
      {
        type: "url",
        href: "https://example.com/help",
        content: "https://example.com/help",
        target: undefined,
        rel: undefined,
      },
    ]);
  });

  it("supports extended attribute syntax for target and rel", () => {
    const result = splitTextAndUrls(
      "Open [in same tab](https://example.com){target=\"_self\"} or [in new](https://example.com){target=\"_blank\" rel=\"nofollow\"}",
    );
    expect(result).toEqual([
      { type: "text", content: "Open " },
      {
        type: "url",
        href: "https://example.com",
        content: "in same tab",
        target: "_self",
        rel: undefined,
      },
      { type: "text", content: " or " },
      {
        type: "url",
        href: "https://example.com",
        content: "in new",
        target: "_blank",
        rel: "nofollow",
      },
    ]);
  });

  it("ignores non-target/rel attributes in extended syntax to prevent injection", () => {
    const result = splitTextAndUrls(
      "Click [safe](https://example.com){target=\"_blank\" onclick=\"alert(1)\" class=\"evil\"}",
    );
    expect(result).toEqual([
      { type: "text", content: "Click " },
      {
        type: "url",
        href: "https://example.com",
        content: "safe",
        target: "_blank",
        rel: "noopener noreferrer",
      },
    ]);
  });

  it("supports hashes and query params in markdown links", () => {
    const result = splitTextAndUrls(
      "Jump to [Top](#top){target=\"_self\"} or filter [30 days](?range=30d){target=\"_self\"}",
    );
    expect(result).toEqual([
      { type: "text", content: "Jump to " },
      {
        type: "url",
        href: "#top",
        content: "Top",
        target: "_self",
        rel: undefined,
      },
      { type: "text", content: " or filter " },
      {
        type: "url",
        href: "?range=30d",
        content: "30 days",
        target: "_self",
        rel: undefined,
      },
    ]);
  });

  it("supports relative paths in markdown links", () => {
    const result = splitTextAndUrls(
      "Go to [Sales](/dashboards/sales){target=\"_self\"} or [Sibling](./other)",
    );
    expect(result).toEqual([
      { type: "text", content: "Go to " },
      {
        type: "url",
        href: "/dashboards/sales",
        content: "Sales",
        target: "_self",
        rel: undefined,
      },
      { type: "text", content: " or " },
      {
        type: "url",
        href: "./other",
        content: "Sibling",
        target: undefined,
        rel: undefined,
      },
    ]);
  });

  it("supports mailto links in markdown syntax", () => {
    const result = splitTextAndUrls("Please [email us](mailto:team@taleshape.com).");
    expect(result).toEqual([
      { type: "text", content: "Please " },
      {
        type: "url",
        href: "mailto:team@taleshape.com",
        content: "email us",
        target: undefined,
        rel: undefined,
      },
      { type: "text", content: "." },
    ]);
  });

  it("leaves unsafe markdown links as plain text", () => {
    const result = splitTextAndUrls(
      "Click [here](javascript:alert(1)){target=\"_blank\"} and [evil](//evil.com)",
    );
    expect(result).toEqual([
      {
        type: "text",
        content: "Click [here](javascript:alert(1)){target=\"_blank\"} and [evil](//evil.com)",
      },
    ]);
  });

  it("handles line breaks alongside links", () => {
    const input = "First line: https://first.com\nSecond line: [second](https://second.com)";
    const result = splitTextAndUrls(input);
    expect(result).toEqual([
      { type: "text", content: "First line: " },
      {
        type: "url",
        href: "https://first.com",
        content: "https://first.com",
        target: "_blank",
        rel: "noopener noreferrer",
      },
      { type: "linebreak", content: "\n" },
      { type: "text", content: "Second line: " },
      {
        type: "url",
        href: "https://second.com",
        content: "second",
        target: undefined,
        rel: undefined,
      },
    ]);
  });
});
