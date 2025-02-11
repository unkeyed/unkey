import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { Refresh3 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useState } from "react";

type RefreshButtonProps = {
  onRefresh: () => void;
  isEnabled: boolean;
  isLive: boolean;
  toggleLive: (value: boolean) => void;
};

const REFRESH_TIMEOUT_MS = 1000;

export const RefreshButton = ({ onRefresh, isEnabled, isLive, toggleLive }: RefreshButtonProps) => {
  const [isLoading, setIsLoading] = useState(false);
  const [refreshTimeout, setRefreshTimeout] = useState<NodeJS.Timeout | null>(null);

  useKeyboardShortcut("r", () => {
    isEnabled && handleRefresh();
  });

  const handleRefresh = () => {
    if (isLoading) {
      return;
    }

    const isLiveBefore = Boolean(isLive);
    setIsLoading(true);
    toggleLive(false);
    onRefresh();

    if (refreshTimeout) {
      clearTimeout(refreshTimeout);
    }

    const timeout = setTimeout(() => {
      setIsLoading(false);
      if (isLiveBefore) {
        toggleLive(true);
      }
    }, REFRESH_TIMEOUT_MS);
    setRefreshTimeout(timeout);
  };

  return (
    <Button
      onClick={handleRefresh}
      variant="ghost"
      title="Refresh data (Shortcut: R)"
      className={cn(
        "px-2 text-accent-12 [&_svg]:text-accent-9",
        "relative overflow-hidden",
        "transition-transform",
        isLoading && "bg-gray-3",
      )}
      disabled={!isEnabled || isLoading}
    >
      {isLoading && <div className="absolute inset-0 bg-accent-6 animate-fill-left" />}
      <Refresh3 className="size-4 relative z-10" />
      <span className="font-medium text-[13px] relative z-10">Refresh</span>
    </Button>
  );
};
