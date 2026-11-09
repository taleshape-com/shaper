// SPDX-License-Identifier: MPL-2.0

import React from "react";
import { splitTextAndUrls, FAST_CHECK_REGEX, isRelativeOrQueryUrl } from "../lib/linkUtils";
import { getRenderMode } from "../lib/utils";

// Component to render text with clickable links and line breaks
export const TextWithLinks: React.FC<{ text?: string | null; className?: string }> = ({ text, className }) => {
  if (!text) {
    return null;
  }

  if (!FAST_CHECK_REGEX.test(text)) {
    return <span className={className}>{text}</span>;
  }

  const parts = splitTextAndUrls(text);
  const isPdf = getRenderMode() === "pdf";

  return (
    <span className={className}>
      {parts.map((part, index) => {
        if (part.type === "url" && part.href) {
          if (isPdf && isRelativeOrQueryUrl(part.href)) {
            return <span key={index}>{part.content}</span>;
          }
          return (
            <a
              key={index}
              href={part.href}
              target={part.target}
              rel={part.rel}
              onClick={(e) => {
                e.stopPropagation();
              }}
              className="shaper-inline-link hover:text-cprimary hover:dark:text-dprimary underline break-words transition-colors"
            >
              {part.content}
            </a>
          );
        }
        if (part.type === "linebreak") {
          return <br key={index} />;
        }
        return <span key={index}>{part.content}</span>;
      })}
    </span>
  );
};

export default TextWithLinks;
