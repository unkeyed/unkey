"use client";

import { useRuntimeLogs } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/logs/context/runtime-logs-provider";
import { LiveSwitchButton } from "@/components/logs/live-switch-button";

export const RuntimeLogsLiveSwitch = () => {
  const { isLive, toggleLive, refresh } = useRuntimeLogs();

  const handleSwitch = () => {
    toggleLive();
    // Leaving Live: re-anchor the historical window to now so rows that streamed
    // in during Live are included. Goes through the refresh signal, leaving the
    // user's existing filters untouched instead of injecting a time filter.
    if (isLive) {
      refresh();
    }
  };

  return <LiveSwitchButton onToggle={handleSwitch} isLive={isLive} />;
};
