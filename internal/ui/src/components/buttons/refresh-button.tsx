"use client";
import { Refresh3 } from "@unkey/icons";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import { useState } from "react";
import { useKeyboardShortcut } from "../../hooks/use-keyboard-shortcut";
import { InfoTooltip } from "../info-tooltip";
import { Button } from "./button";
import { KeyboardButton } from "./keyboard-button";

type RefreshButtonProps = {
  onRefresh: () => void;
  isEnabled: boolean;
  isLive?: boolean;
  toggleLive?: (value: boolean) => void;
};

const REFRESH_TIMEOUT_MS = 1000;

const RefreshButton = ({ onRefresh, isEnabled, isLive, toggleLive }: RefreshButtonProps) => {
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

  useKeyboardShortcut("option+shift+w", handleRefresh, {
    preventDefault: true,
    disabled: !isEnabled,
  });

  return (
    <InfoTooltip
      content="Refresh unavailable - please select a relative time filter in the 'Since' dropdown"
      variant="inverted"
      position={{ side: "bottom", align: "center" }}
      disabled={isEnabled && !isLoading}
      asChild
    >
      <div>
        <Button
          onClick={handleRefresh}
          variant="ghost"
          size="md"
          title={isEnabled ? "Refresh data (Shortcut: ⌥+⇧+W)" : ""}
          disabled={!isEnabled || isLoading}
          loading={isLoading}
          className="flex w-full items-center justify-center rounded-lg border border-gray-4"
        >
          <Refresh3 className="size-4" />
          <span className="font-medium text-[13px] relative z-10">Refresh</span>
          <KeyboardButton shortcut="⌥+⇧+W" />
        </Button>
      </div>
    </InfoTooltip>
  );
};

RefreshButton.displayName = "RefreshButton";
export { RefreshButton, type RefreshButtonProps };
