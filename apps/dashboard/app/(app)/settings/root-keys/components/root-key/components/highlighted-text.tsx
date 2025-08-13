"use client";

import type React from "react";

type HighlightedTextProps = {
  text: string;
  searchValue: string | undefined;
};

export function HighlightedText({ text, searchValue }: HighlightedTextProps): React.ReactNode {
  const query = searchValue?.trim() ?? "";
  if (query === "") {
    return text;
  }

  const escapedSearchValue = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  // Add 'u' for better Unicode handling (e.g., case-insensitive matching across locales)
  const regex = new RegExp(`(${escapedSearchValue})`, "giu");
  const nonGlobalRegex = new RegExp(regex.source, regex.flags.replace("g", ""));
  const parts = text.split(regex);

  return parts.map((part, index) =>
    nonGlobalRegex.test(part) ? (
      <span key={index + part} className="bg-grayA-4 rounded-[4px] py-0.5">
        {part}
      </span>
    ) : (
      part
    ),
  );
}
