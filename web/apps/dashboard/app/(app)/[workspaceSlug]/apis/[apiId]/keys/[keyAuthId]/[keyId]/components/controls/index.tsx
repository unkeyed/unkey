"use client";
import { StatusBadge } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/settings/components/status-badge";
import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { Presence } from "@/components/presence";
import { formatNumber } from "@/lib/fmt";
import { queryKeys } from "@/lib/query-keys";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useQuery } from "@tanstack/react-query";
import { IconCoinsOutline18 } from "@unkey/icons";
import { Separator } from "@unkey/ui";
import { cn } from "cn";
import { LogsDateTime } from "./components/logs-datetime";
import { LogsFilters } from "./components/logs-filters";
import { LogsLiveSwitch } from "./components/logs-live-switch";
import { LogsRefresh } from "./components/logs-refresh";
import { LogsSearch } from "./components/logs-search";

export function KeysDetailsLogsControls({
  keyId,
  apiId,
}: {
  keyId: string;
  apiId: string;
}) {
  const { data } = useQuery({
    queryKey: queryKeys.keys.detail(keyId),
    queryFn: () => getUnkeyClient().keys.getKey({ keyId }),
  });
  const remainingCredit = data?.data.credits?.remaining;
  const hasRemainingCredit = remainingCredit !== null && remainingCredit !== undefined;
  return (
    <ControlsContainer>
      <ControlsLeft>
        <LogsSearch apiId={apiId} />
        <LogsFilters />
        <LogsDateTime />
        <Presence present={hasRemainingCredit} exitMs={300}>
          {({ closing }) => (
            <div
              className={cn(
                "flex items-center transition-[opacity,translate] duration-300 ease-[ease-out]",
                closing
                  ? "opacity-0 -translate-x-[5px]"
                  : "starting:opacity-0 starting:-translate-x-[5px]",
              )}
            >
              <Separator
                orientation="vertical"
                className="flex items-center justify-center h-4 mx-1 my-auto"
              />
              <div className="items-center flex justify-center gap-2">
                <div className="text-gray-12 font-medium text-sm max-md:hidden pl-4 transition-opacity delay-50 duration-200 ease-[ease-out] starting:opacity-0">
                  Remaining Credits:
                </div>
                <div
                  key={(remainingCredit ?? 0) > 0 ? "enabled" : "disabled"}
                  className="transition-[opacity,scale] delay-100 duration-200 ease-[ease-out] starting:opacity-0 starting:scale-[0.97]"
                >
                  {(remainingCredit ?? 0) > 0 ? (
                    <StatusBadge
                      className="text-xs"
                      variant="enabled"
                      text={formatNumber(remainingCredit ?? 0)}
                      icon={<IconCoinsOutline18 className="size-3" />}
                    />
                  ) : (
                    <StatusBadge
                      className="text-xs"
                      variant="disabled"
                      text="0"
                      icon={<IconCoinsOutline18 className="size-3" />}
                    />
                  )}
                </div>
              </div>
            </div>
          )}
        </Presence>
      </ControlsLeft>
      <ControlsRight>
        <LogsLiveSwitch />
        <LogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
