import { XMark } from "@unkey/icons";
import type React from "react";
import { SearchExampleTooltip } from "./search-example-tooltip";

type SearchActionsProps = {
  exampleQueries?: string[];
  searchText: string;
  hideClear: boolean;
  hideExplainer: boolean;
  isProcessing: boolean;
  searchMode: "allowTypeDuringSearch" | "debounced" | "manual";
  onClear: () => void;
  onSelectExample: (query: string) => void;
};

/**
 * SearchActions component renders the right-side actions (clear button or examples tooltip)
 */
export const SearchActions: React.FC<SearchActionsProps> = ({
  exampleQueries,
  searchText,
  hideClear,
  hideExplainer,
  isProcessing,
  searchMode,
  onClear,
  onSelectExample,
}) => {
  // Don't render anything if processing (unless in allowTypeDuringSearch mode)
  if (isProcessing && searchMode !== "allowTypeDuringSearch") {
    return null;
  }

  // Render clear button when there's text
  if (searchText.length > 0 && !hideClear) {
    return (
      <button
        aria-label="Clear search"
        onClick={onClear}
        type="button"
        data-testid="clear-search-button"
      >
        <XMark className="size-4 text-accent-9" />
      </button>
    );
  }

  if (searchText.length === 0 && !hideExplainer) {
    return (
      <SearchExampleTooltip onSelectExample={onSelectExample} exampleQueries={exampleQueries} />
    );
  }

  return null;
};
