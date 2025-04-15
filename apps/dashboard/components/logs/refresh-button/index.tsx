import { RatelimitOverviewTooltip } from "@/app/(app)/ratelimits/[namespaceId]/_overview/components/table/components/ratelimit-overview-tooltip";
import { KeyboardButton } from "@/components/keyboard-button";
import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { Refresh3 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";

type RefreshButtonProps = {
  onRefresh: () => void;
  isEnabled: boolean;
  isLive?: boolean;
  toggleLive?: (value: boolean) => void;
};

const REFRESH_TIMEOUT_MS = 1000;

export const RefreshButton = ({ onRefresh, isEnabled, isLive, toggleLive }: RefreshButtonProps) => {
  const [isLoading, setIsLoading] = useState(false);
  const [refreshTimeout, setRefreshTimeout] = useState<NodeJS.Timeout | null>(null);

  const handleRefresh = () => {
    if (isLoading) {
      return;
    }
    const isLiveBefore = Boolean(isLive);
    setIsLoading(true);
    toggleLive?.(false);
    onRefresh();

    if (refreshTimeout) {
      clearTimeout(refreshTimeout);
    }

    const timeout = setTimeout(() => {
      setIsLoading(false);
      if (isLiveBefore) {
        toggleLive?.(true);
      }
    }, REFRESH_TIMEOUT_MS);

    setRefreshTimeout(timeout);
  };

  useKeyboardShortcut("option+shift+r", handleRefresh, {
    preventDefault: true,
    disabled: !isEnabled,
  });

  return (
    <RatelimitOverviewTooltip
      content="Refresh unavailable - please select a relative time filter in the 'Since' dropdown"
      position={{ side: "bottom", align: "center" }}
      disabled={isEnabled && !isLoading}
      asChild
    >
      <div>
        <Button
          onClick={handleRefresh}
          variant="ghost"
          size="md"
          title={isEnabled ? "Refresh data (Shortcut: ⌥+⇧+P)" : ""}
          disabled={!isEnabled || isLoading}
          loading={isLoading}
          className="flex w-full items-center justify-center rounded-lg border border-gray-4"
        >
          <Refresh3 className="size-4" />
          <span className="font-medium text-[13px] relative z-10">Refresh</span>
          <KeyboardButton shortcut="⌥+⇧+R" />
        </Button>
      </div>
    </RatelimitOverviewTooltip>
  );
};
