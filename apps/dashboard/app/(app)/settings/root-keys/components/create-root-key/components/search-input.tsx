"use client";
import { Input } from "@unkey/ui";
import { cn } from "lib/utils";

type SearchInputProps = {
  maxLength?: number;
  className?: string;
  value: string;
  placeholder: string;
  isProcessing: boolean;
  isLoading: boolean;
  loadingText: string;
  clearingText: string;
  searchMode: "allowTypeDuringSearch" | "debounced" | "manual";
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void;
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
  onKeyDown,
  inputRef,
}: SearchInputProps) => {
  // Show loading state unless we're in allowTypeDuringSearch mode
  if (isProcessing && searchMode !== "allowTypeDuringSearch") {
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
      onKeyDown={onKeyDown}
      onChange={onChange}
      maxLength={maxLength}
      placeholder={placeholder}
      className={cn(
        "truncate text-accent-12 font-medium text-[13px] bg-transparent border-none outline-none focus:ring-0 focus:outline-none placeholder:text-accent-8 selection:bg-gray-6 w-full",
        className,
      )}
      disabled={isProcessing && searchMode !== "allowTypeDuringSearch"}
      data-testid="search-input"
    />
  );
};
