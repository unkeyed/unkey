"use client";
import { StatusBadge } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/settings/components/status-badge";
import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { OPTIONS_WITHOUT_MINUTES } from "@/components/logs/datetime/constants";
import { MotionProvider } from "@/components/motion-provider";
import { TimeRangeFilter } from "@/components/time-range-filter";
import { formatNumber } from "@/lib/fmt";
import { queryKeys } from "@/lib/query-keys";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useQuery } from "@tanstack/react-query";
import { IconCoinsOutline18 } from "@unkey/icons";
import { Separator } from "@unkey/ui";
import { AnimatePresence, m } from "framer-motion";
import { useFilters } from "../../hooks/use-filters";
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
  const { filters, updateFilters } = useFilters();
  const hasRemainingCredit = remainingCredit !== null && remainingCredit !== undefined;

  return (
    <MotionProvider>
      <ControlsContainer>
        <ControlsLeft>
          <LogsSearch apiId={apiId} />
          <LogsFilters />
          <TimeRangeFilter
            filters={filters}
            updateFilters={updateFilters}
            look="logsCompact"
            customOptions={OPTIONS_WITHOUT_MINUTES}
          />
          <AnimatePresence mode="wait">
            {hasRemainingCredit ? (
              <m.div
                key="remaining-credit"
                className="flex items-center"
                initial={{ opacity: 0, x: -5 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -5 }}
                transition={{
                  duration: 0.3,
                  ease: "easeOut",
                }}
              >
                <Separator
                  orientation="vertical"
                  className="flex items-center justify-center h-4 mx-1 my-auto"
                />
                <div className="items-center flex justify-center gap-2">
                  <m.div
                    className="text-gray-12 font-medium text-sm max-md:hidden pl-4"
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    transition={{ delay: 0.05, duration: 0.2 }}
                  >
                    Remaining Credits:
                  </m.div>
                  {(remainingCredit ?? 0) > 0 ? (
                    <m.div
                      initial={{ opacity: 0, scale: 0.97 }}
                      animate={{ opacity: 1, scale: 1 }}
                      transition={{
                        delay: 0.1,
                        duration: 0.2,
                        scale: {
                          type: "spring",
                          stiffness: 500,
                          damping: 25,
                        },
                      }}
                    >
                      <StatusBadge
                        className="text-xs"
                        variant="enabled"
                        text={formatNumber(remainingCredit ?? 0)}
                        icon={<IconCoinsOutline18 className="size-3" />}
                      />
                    </m.div>
                  ) : (
                    <m.div
                      initial={{ opacity: 0, scale: 0.97 }}
                      animate={{ opacity: 1, scale: 1 }}
                      transition={{
                        delay: 0.1,
                        duration: 0.2,
                        scale: {
                          type: "spring",
                          stiffness: 500,
                          damping: 25,
                        },
                      }}
                    >
                      <StatusBadge
                        className="text-xs"
                        variant="disabled"
                        text="0"
                        icon={<IconCoinsOutline18 className="size-3" />}
                      />
                    </m.div>
                  )}
                </div>
              </m.div>
            ) : null}
          </AnimatePresence>
        </ControlsLeft>
        <ControlsRight>
          <LogsLiveSwitch />
          <LogsRefresh />
        </ControlsRight>
      </ControlsContainer>
    </MotionProvider>
  );
}
