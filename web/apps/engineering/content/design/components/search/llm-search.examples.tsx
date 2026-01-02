"use client";

import { RenderComponentWithSnippet } from "@/app/components/render";
import { LLMSearch } from "@unkey/ui";
import { useCallback, useState } from "react";

// Types
interface SearchExampleProps {
  children: React.ReactNode;
  className?: string;
}

interface UseSearchStateOptions {
  delay?: number;
  onSearch?: (query: string) => void;
  onClear?: () => void;
}

// Custom hooks
function useSearchState({ delay = 800, onSearch, onClear }: UseSearchStateOptions = {}) {
  const [isLoading, setIsLoading] = useState(false);

  const handleSearch = useCallback(
    (query: string) => {
      setIsLoading(true);
      onSearch?.(query);
      setTimeout(() => setIsLoading(false), delay);
    },
    [delay, onSearch],
  );

  const handleClear = useCallback(() => {
    onClear?.();
  }, [onClear]);

  return { isLoading, handleSearch, handleClear };
}

function useSearchWithResults() {
  const [searchResults, setSearchResults] = useState<string[]>([]);

  const handleSearch = useCallback((query: string) => {
    setSearchResults([`Results for: "${query}"`]);
  }, []);

  const handleClear = useCallback(() => {
    setSearchResults([]);
  }, []);

  return { searchResults, handleSearch, handleClear };
}

// Reusable components
function SearchExampleWrapper({ children, className = "w-full max-w-md" }: SearchExampleProps) {
  return (
    <RenderComponentWithSnippet>
      <div className={className}>{children}</div>
    </RenderComponentWithSnippet>
  );
}

function SearchResults({ results }: { results: string[] }) {
  if (results.length === 0) {
    return null;
  }

  return (
    <div className="mt-4 p-3 bg-gray-50 rounded-md">
      <h4 className="text-sm font-medium mb-2">Search Results:</h4>
      <ul className="text-sm space-y-1">
        {results.map((result) => (
          <li key={result}>{result}</li>
        ))}
      </ul>
    </div>
  );
}

// Example configurations
const EXAMPLE_QUERIES = {
  default: [
    "Show me errors from the last hour",
    "Find requests from user ID 12345",
    "Display API calls with status 500",
  ],
  logs: [
    "What's causing the high latency?",
    "Show me all authentication failures",
    "Find requests from mobile devices",
  ],
};

// Example components
export function DefaultLLMSearch() {
  const { searchResults, handleSearch, handleClear } = useSearchWithResults();
  const { isLoading, handleSearch: handleSearchWithLoading } = useSearchState({
    delay: 1000,
    onSearch: handleSearch,
  });

  return (
    <SearchExampleWrapper className="flex flex-col gap-4 w-full max-w-md">
      <LLMSearch
        onSearch={handleSearchWithLoading}
        onClear={handleClear}
        isLoading={isLoading}
        exampleQueries={EXAMPLE_QUERIES.default}
      />
      <SearchResults results={searchResults} />
    </SearchExampleWrapper>
  );
}

export function LLMSearchWithCustomPlaceholder() {
  const { isLoading, handleSearch } = useSearchState({
    delay: 800,
    onSearch: (_query) => {},
  });

  return (
    <SearchExampleWrapper>
      <LLMSearch
        onSearch={handleSearch}
        isLoading={isLoading}
        placeholder="Ask me anything about your logs..."
        exampleQueries={EXAMPLE_QUERIES.logs}
      />
    </SearchExampleWrapper>
  );
}

export function LLMSearchWithDebouncedMode() {
  const [lastQuery, setLastQuery] = useState("");
  const { isLoading, handleSearch } = useSearchState({
    delay: 500,
    onSearch: setLastQuery,
  });

  return (
    <SearchExampleWrapper className="flex flex-col gap-4 w-full max-w-md">
      <LLMSearch
        onSearch={handleSearch}
        isLoading={isLoading}
        searchMode="debounced"
        debounceTime={300}
        placeholder="Type to search with debouncing..."
      />
      {lastQuery && <div className="text-sm text-gray-600">Last search: "{lastQuery}"</div>}
    </SearchExampleWrapper>
  );
}

export function LLMSearchWithThrottledMode() {
  const [searchCount, setSearchCount] = useState(0);
  const { isLoading, handleSearch } = useSearchState({
    delay: 400,
    onSearch: () => setSearchCount((prev) => prev + 1),
  });

  return (
    <SearchExampleWrapper className="flex flex-col gap-4 w-full max-w-md">
      <LLMSearch
        onSearch={handleSearch}
        isLoading={isLoading}
        searchMode="allowTypeDuringSearch"
        placeholder="Type to search with throttling..."
      />
      <div className="text-sm text-gray-600">Search count: {searchCount}</div>
    </SearchExampleWrapper>
  );
}

export function LLMSearchWithCustomTexts() {
  const { isLoading, handleSearch } = useSearchState({ delay: 1200 });

  return (
    <SearchExampleWrapper>
      <LLMSearch
        onSearch={handleSearch}
        isLoading={isLoading}
        loadingText="Analyzing your logs with AI..."
        clearingText="Clearing search results..."
        placeholder="Search your application logs..."
      />
    </SearchExampleWrapper>
  );
}

export function LLMSearchWithoutExplainer() {
  const { isLoading, handleSearch } = useSearchState({ delay: 800 });

  return (
    <SearchExampleWrapper>
      <LLMSearch
        onSearch={handleSearch}
        isLoading={isLoading}
        hideExplainer={true}
        placeholder="Search logs..."
      />
    </SearchExampleWrapper>
  );
}

export function LLMSearchWithoutClear() {
  const { isLoading, handleSearch } = useSearchState({ delay: 800 });

  return (
    <SearchExampleWrapper>
      <LLMSearch
        onSearch={handleSearch}
        isLoading={isLoading}
        hideClear={true}
        placeholder="Search logs..."
      />
    </SearchExampleWrapper>
  );
}

export function LLMSearchWithKeyboardShortcuts() {
  const [lastAction, setLastAction] = useState("");
  const { isLoading, handleSearch, handleClear } = useSearchState({
    delay: 600,
    onSearch: (query) => setLastAction(`Searched: "${query}"`),
    onClear: () => setLastAction("Cleared search"),
  });

  return (
    <SearchExampleWrapper className="flex flex-col gap-4 w-full max-w-md">
      <LLMSearch
        onSearch={handleSearch}
        onClear={handleClear}
        isLoading={isLoading}
        placeholder="Press 'S' to focus, Enter to search, Esc to clear..."
      />
      {lastAction && <div className="text-sm text-gray-600">Last action: {lastAction}</div>}
      <div className="text-xs text-gray-500 space-y-1">
        <div>Keyboard shortcuts:</div>
        <div>• Press 'S' to focus the search</div>
        <div>• Press 'Enter' to search</div>
        <div>• Press 'Esc' to clear</div>
      </div>
    </SearchExampleWrapper>
  );
}
