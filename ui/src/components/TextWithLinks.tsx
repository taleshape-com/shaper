// SPDX-License-Identifier: MPL-2.0

import React from "react";

// URL detection regex pattern
const URL_REGEX = /(https?:\/\/[^\s]+)/g;

// Function to split text into parts (text, URLs, and line breaks)
const splitTextAndUrls = (text: string): Array<{ type: 'text' | 'url' | 'linebreak', content: string }> => {
	const parts: Array<{ type: 'text' | 'url' | 'linebreak', content: string }> = [];
	let lastIndex = 0;
	let match;

	// Reset regex state
	URL_REGEX.lastIndex = 0;

	while ((match = URL_REGEX.exec(text)) !== null) {
		// Add text before URL (handle line breaks within it)
		if (match.index > lastIndex) {
			const textBefore = text.slice(lastIndex, match.index);
			parts.push(...splitTextByLineBreaks(textBefore));
		}

		// Add URL
		parts.push({
			type: 'url',
			content: match[0]
		});

		lastIndex = match.index + match[0].length;
	}

	// Add remaining text (handle line breaks within it)
	if (lastIndex < text.length) {
		const remainingText = text.slice(lastIndex);
		parts.push(...splitTextByLineBreaks(remainingText));
	}

	return parts;
};

// Helper function to split text by line breaks
const splitTextByLineBreaks = (text: string): Array<{ type: 'text' | 'linebreak', content: string }> => {
	const parts: Array<{ type: 'text' | 'linebreak', content: string }> = [];
	const segments = text.split('\n');

	for (let i = 0; i < segments.length; i++) {
		if (segments[i]) {
			parts.push({
				type: 'text',
				content: segments[i]
			});
		}

		// Add line break between segments (except after the last one)
		if (i < segments.length - 1) {
			parts.push({
				type: 'linebreak',
				content: '\n'
			});
		}
	}

	return parts;
};

const removeLinkPrefix = (link: string): string => {
	return link.replace(/^https?:\/\//, '');
};

// Function to check if text contains URLs or line breaks
const containsUrlsOrLineBreaks = (text: string): boolean => {
	return URL_REGEX.test(text) || text.includes('\n');
};

// Component to render text with clickable links and line breaks
export const TextWithLinks: React.FC<{ text: string; className?: string }> = ({ text, className }) => {
	if (!containsUrlsOrLineBreaks(text)) {
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
				if (part.type === 'linebreak') {
					return <br key={index} />;
				}
				return <span key={index}>{part.content}</span>;
			})}
		</span>
	);
};

export default TextWithLinks;