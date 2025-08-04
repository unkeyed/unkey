"use client";

import type React from "react";

type HighlightedTextProps = {
  text: string;
  searchValue: string | undefined;
};

export function HighlightedText({ text, searchValue }: HighlightedTextProps): React.ReactNode {
  if (!searchValue || searchValue.trim() === "") {
    return text;
  }

  const escapedSearchValue = searchValue.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const regex = new RegExp(`(${escapedSearchValue})`, "gi");
  const parts = text.split(regex);

  return parts.map((part, index) =>
    regex.test(part) ? (
      <span key={index + part} className="bg-grayA-4 rounded-[4px] py-0.5">
        {part}
      </span>
    ) : (
      part
    ),
  );
}
