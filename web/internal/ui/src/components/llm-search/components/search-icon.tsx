import { IconMagnifierOutline18, IconRefresh3Outline18 } from "nucleo-ui-outline-18";

type SearchIconProps = {
  isProcessing: boolean;
};

export const SearchIcon = ({ isProcessing }: SearchIconProps) => {
  if (isProcessing) {
    return (
      <IconRefresh3Outline18
        className="text-accent-10 size-4 animate-spin"
        data-testid="loading-icon"
      />
    );
  }

  return <IconMagnifierOutline18 className="text-accent-9 size-4" data-testid="search-icon" />;
};
