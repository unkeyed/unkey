"use client";
import { cn } from "@/lib/utils";
import { Input } from "@unkey/ui";

export const SEARCH_MODES = {
  ALLOW_TYPE: "allowTypeDuringSearch",
  DEBOUNCED: "debounced",
  MANUAL: "manual",
} as const;

type SearchInputProps = {
  maxLength?: number;
  className?: string;
  value: string;
  placeholder: string;
  isProcessing: boolean;
  isLoading: boolean;
  loadingText: string;
  clearingText: string;
  searchMode: (typeof SEARCH_MODES)[keyof typeof SEARCH_MODES];
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  inputRef: React.RefObject<HTMLInputElement>;
};

export const SearchInput = ({
  maxLength,
  className,
  value,
  placeholder,
  isProcessing,
  isLoading,
  loadingText,
  clearingText,
  searchMode,
  onChange,
  inputRef,
}: SearchInputProps) => {
  if (isProcessing && searchMode !== SEARCH_MODES.ALLOW_TYPE) {
    return (
      <div className="text-accent-11 text-[13px] animate-pulse" data-testid="search-loading-state">
        {isLoading ? loadingText : clearingText}
      </div>
    );
  }

  return (
    <Input
      variant="ghost"
      ref={inputRef}
      type="text"
      value={value}
      onChange={onChange}
      placeholder={placeholder}
      maxLength={maxLength}
      className={cn(
        "truncate w-full focus:ring-0 focus:outline-none focus:border-none selection:border-none selection:ring-0 ring-0 border-none",
        className,
      )}
      disabled={isProcessing && searchMode !== SEARCH_MODES.ALLOW_TYPE}
      data-testid="search-input"
    />
  );
};
