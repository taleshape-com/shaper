// SPDX-License-Identifier: MPL-2.0

// Safe URL check: only allow https://, http://, mailto:, relative paths (/..., ./..., ../...), hashes (#...), and query params (?...)
export const isSafeUrl = (url: string): boolean => {
  if (!url) return false;

  const trimmed = url.trim();
  if (!trimmed) return false;

  // Disallow control characters and whitespace
  for (let i = 0; i < trimmed.length; i++) {
    const code = trimmed.charCodeAt(i);
    if (code <= 32 || code === 127) {
      return false;
    }
  }

  // Reject protocol-relative URLs (e.g. //evil.com)
  if (trimmed.startsWith("//")) return false;

  // Safe relative paths, hashes, and query params
  if (
    trimmed.startsWith("/") ||
    trimmed.startsWith("./") ||
    trimmed.startsWith("../") ||
    trimmed.startsWith("#") ||
    trimmed.startsWith("?")
  ) {
    return true;
  }

  // Validate protocols using WHATWG URL parser
  try {
    const parsed = new URL(trimmed);
    return (
      (parsed.protocol === "https:" ||
        parsed.protocol === "http:" ||
        parsed.protocol === "mailto:") &&
      Boolean(parsed.hostname || parsed.protocol === "mailto:")
    );
  } catch {
    return false;
  }
};

// Check if a URL is relative or query-only (e.g. /..., ./..., ../..., ?...)
// Note: In-page fragment/hash links (#...) are excluded because Chromium converts them
// into native internal PDF destinations (/Dest) that jump to sections within the PDF.
export const isRelativeOrQueryUrl = (url: string): boolean => {
  if (!url) return false;
  const trimmed = url.trim();
  return (
    trimmed.startsWith("/") ||
    trimmed.startsWith("./") ||
    trimmed.startsWith("../") ||
    trimmed.startsWith("?")
  );
};

// Parses extended markdown attributes e.g. {target="_blank" rel="noopener"}
// Strictly limited to `target` and `rel` attributes; ignores all other attributes.
export const parseLinkAttributes = (
  attrsString?: string,
): { target?: string; rel?: string } => {
  if (!attrsString) {
    return {};
  }

  const ATTR_REGEX = /\b(target|rel)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'=<>`]+))/gi;
  let target: string | undefined;
  let rel: string | undefined;
  let match: RegExpExecArray | null;

  while ((match = ATTR_REGEX.exec(attrsString)) !== null) {
    const key = match[1].toLowerCase();
    const rawVal = match[2] ?? match[3] ?? match[4] ?? "";
    if (key === "target") {
      if (/^(_blank|_self|_parent|_top|[a-zA-Z0-9_-]+)$/.test(rawVal)) {
        target = rawVal;
      }
    } else if (key === "rel") {
      if (/^[a-zA-Z0-9_ -]+$/.test(rawVal)) {
        rel = rawVal.trim();
      }
    }
  }

  let finalRel = rel;
  if (finalRel === undefined && target === "_blank") {
    finalRel = "noopener noreferrer";
  }

  return { target, rel: finalRel };
};

// Strips common trailing punctuation and unbalanced closing brackets from bare URLs
export const trimBareUrl = (rawUrl: string): { url: string; trailing: string } => {
  let url = rawUrl;
  let trailing = "";
  const trailingPunctuation = /[.,:;!?'"”’>]$/;

  while (url.length > 0) {
    if (trailingPunctuation.test(url)) {
      trailing = url.slice(-1) + trailing;
      url = url.slice(0, -1);
      continue;
    }
    // Check if ending with ')' and parens in url are unbalanced
    if (url.endsWith(")")) {
      const openParens = (url.match(/\(/g) || []).length;
      const closeParens = (url.match(/\)/g) || []).length;
      if (closeParens > openParens) {
        trailing = url.slice(-1) + trailing;
        url = url.slice(0, -1);
        continue;
      }
    }
    // Check if ending with ']' and brackets in url are unbalanced
    if (url.endsWith("]")) {
      const openBrackets = (url.match(/\[/g) || []).length;
      const closeBrackets = (url.match(/\]/g) || []).length;
      if (closeBrackets > openBrackets) {
        trailing = url.slice(-1) + trailing;
        url = url.slice(0, -1);
        continue;
      }
    }
    break;
  }

  return { url, trailing };
};

// Tokenizer pattern: newlines, markdown links [text](url){attrs}, or bare URLs
// Handles balanced single-level parens inside markdown URLs like (https://en.wikipedia.org/wiki/Go_(programming_language))
const TOKEN_REGEX = /(\n)|\[([^\]]*)\]\(((?:[^\s()]|\([^\s()]+\))+)\)(?:\{([^{}]+)\})?|(https?:\/\/[^\s<]+)/g;

export const FAST_CHECK_REGEX = /https?:\/\/|\[[^\]]*\]\([^)\s]+\)|\n/;

export interface TextPart {
  type: "text" | "url" | "linebreak";
  content: string;
  href?: string;
  target?: string;
  rel?: string;
}

export const splitTextAndUrls = (text: string): TextPart[] => {
  const parts: TextPart[] = [];
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  const appendText = (content: string) => {
    if (!content) return;
    const last = parts[parts.length - 1];
    if (last && last.type === "text") {
      last.content += content;
    } else {
      parts.push({ type: "text", content });
    }
  };

  TOKEN_REGEX.lastIndex = 0;

  while ((match = TOKEN_REGEX.exec(text)) !== null) {
    if (match.index > lastIndex) {
      appendText(text.slice(lastIndex, match.index));
    }

    if (match[1]) {
      // Line break
      parts.push({ type: "linebreak", content: "\n" });
    } else if (match[2] !== undefined && match[3] !== undefined) {
      // Markdown link: [text](url){attrs}
      const linkText = match[2];
      const linkUrl = match[3];
      const attrsStr = match[4];

      if (isSafeUrl(linkUrl)) {
        const { target, rel } = parseLinkAttributes(attrsStr);
        parts.push({
          type: "url",
          href: linkUrl,
          content: linkText || linkUrl,
          target,
          rel,
        });
      } else {
        appendText(match[0]);
      }
    } else if (match[5]) {
      // Bare URL: displayed as is without stripping http/https
      const { url, trailing } = trimBareUrl(match[5]);
      if (isSafeUrl(url)) {
        parts.push({
          type: "url",
          href: url,
          content: url,
          target: "_blank",
          rel: "noopener noreferrer",
        });
        if (trailing) {
          appendText(trailing);
        }
      } else {
        appendText(match[0]);
      }
    }

    lastIndex = match.index + match[0].length;
  }

  if (lastIndex < text.length) {
    appendText(text.slice(lastIndex));
  }

  return parts;
};
