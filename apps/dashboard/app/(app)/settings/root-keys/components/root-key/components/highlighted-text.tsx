"use client";

import { useMemo } from "react";
import type React from "react";

type HighlightedTextProps = {
  text: string;
  searchValue: string | undefined;
};

export function HighlightedText({ text, searchValue }: HighlightedTextProps): React.ReactNode {
  const query = searchValue?.trim() ?? "";

  const parts = useMemo(() => {
    if (query === "") {
      return [text];
    }

    const escapedSearchValue = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    // Add 'u' for better Unicode handling (e.g., case-insensitive matching across locales)
    const regex = new RegExp(`(${escapedSearchValue})`, "iu");
    return text.split(regex);
  }, [text, query]);

  return parts.map((part, index) =>
    index % 2 === 1 ? (
      <mark key={index + part} className="bg-grayA-4 rounded-[4px] py-0.5 text-grayA-12">
        {part}
      </mark>
    ) : (
      part
    ),
  );
}
