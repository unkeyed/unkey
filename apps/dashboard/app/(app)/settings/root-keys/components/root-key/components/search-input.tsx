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
      ref={inputRef}
      type="text"
      value={value}
      onChange={onChange}
      placeholder={placeholder}
      className={cn(
        "truncate text-accent-12 font-medium text-[13px] bg-transparent focus:!bg-transparent border-none outline-none focus:ring-0 focus:outline-none placeholder:text-accent-8 selection:!bg-transparent w-full gap-4",
        className,
      )}
      disabled={isProcessing}
      data-testid="search-input"
    />
  );
};
