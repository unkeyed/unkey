// Duplicate of @unkey/ui/components/llm-search/components/search-icon.tsx
// TODO: Remove this once we have a proper search icon component
import { Magnifier, Refresh3 } from "@unkey/icons";

type SearchIconProps = {
  isProcessing: boolean;
};

export const SearchIcon = ({ isProcessing }: SearchIconProps) => {
  if (isProcessing) {
    return <Refresh3 className="text-accent-10 size-4 animate-spin" data-testid="loading-icon" />;
  }

  return <Magnifier className="text-accent-9 size-4" data-testid="search-icon" />;
};
