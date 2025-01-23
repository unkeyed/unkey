import { trpc } from "@/lib/trpc/client";
import { Refresh3 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useState } from "react";
import { useLogsContext } from "../../../context/logs";
import { useFilters } from "../../../hooks/use-filters";
import { useKeyboardShortcut } from "../../../hooks/use-keyboard-shortcut";

export const LogsRefresh = () => {
  const { toggleLive } = useLogsContext();
  const { filters } = useFilters();
  const { logs } = trpc.useUtils();
  const [isLoading, setIsLoading] = useState(false);

  const hasRelativeFilter = filters.find((f) => f.field === "since");
  useKeyboardShortcut("r", () => {
    hasRelativeFilter && handleSwitch();
  });

  const handleSwitch = () => {
    setIsLoading(true);
    toggleLive(false);
    logs.queryLogs.invalidate();

    setTimeout(() => {
      setIsLoading(false);
    }, 1000);
  };

  return (
    <Button
      onClick={handleSwitch}
      variant="ghost"
      className={cn(
        "px-2 text-accent-12 [&_svg]:text-accent-9",
        "relative overflow-hidden",
        "transition-transform",
        isLoading && "bg-gray-3",
      )}
      disabled={!hasRelativeFilter || isLoading}
    >
      {isLoading && <div className="absolute inset-0 bg-accent-6 animate-fill-left" />}
      <Refresh3 className="size-4 relative z-10" />
      <span className="font-medium text-[13px] relative z-10">Refresh</span>
    </Button>
  );
};
