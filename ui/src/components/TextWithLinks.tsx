// SPDX-License-Identifier: MPL-2.0

import React from "react";

// URL detection regex pattern
const URL_REGEX = /(https?:\/\/[^\s]+)/g;

// Function to detect if a string contains URLs
const containsUrls = (text: string): boolean => {
  return URL_REGEX.test(text);
};

// Function to split text into parts (text and URLs)
const splitTextAndUrls = (text: string): Array<{ type: 'text' | 'url', content: string }> => {
  const parts: Array<{ type: 'text' | 'url', content: string }> = [];
  let lastIndex = 0;
  let match;

  // Reset regex state
  URL_REGEX.lastIndex = 0;
  
  while ((match = URL_REGEX.exec(text)) !== null) {
    // Add text before URL
    if (match.index > lastIndex) {
      parts.push({
        type: 'text',
        content: text.slice(lastIndex, match.index)
      });
    }
    
    // Add URL
    parts.push({
      type: 'url',
      content: match[0]
    });
    
    lastIndex = match.index + match[0].length;
  }
  
  // Add remaining text
  if (lastIndex < text.length) {
    parts.push({
      type: 'text',
      content: text.slice(lastIndex)
    });
  }
  
  return parts;
};

const removeLinkPrefix = (link: string): string => {
  return link.replace(/^https?:\/\//, '');
};

// Component to render text with clickable links
export const TextWithLinks: React.FC<{ text: string; className?: string }> = ({ text, className }) => {
  if (!containsUrls(text)) {
    return <span className={className}>{text}</span>;
  }

  const parts = splitTextAndUrls(text);
  
  return (
    <span className={className}>
      {parts.map((part, index) => {
        if (part.type === 'url') {
          return (
            <a
              key={index}
              href={part.content}
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-cprimary hover:dark:text-dprimary underline break-all transition-colors"
            >
              {removeLinkPrefix(part.content)}
            </a>
          );
        }
        return <span key={index}>{part.content}</span>;
      })}
    </span>
  );
};

export default TextWithLinks; 