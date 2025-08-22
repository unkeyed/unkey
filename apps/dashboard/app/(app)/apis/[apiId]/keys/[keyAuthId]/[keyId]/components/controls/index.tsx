import { StatusBadge } from "@/app/(app)/apis/[apiId]/settings/components/status-badge";
import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { formatRawNumber } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { ChartUsage, Coins } from "@unkey/icons";
import { Separator } from "@unkey/ui";
import { AnimatePresence, motion } from "framer-motion";
import { useSpentCredits } from "../../hooks/use-spent-credits";
import { LogsDateTime } from "./components/logs-datetime";
import { LogsFilters } from "./components/logs-filters";
import { LogsLiveSwitch } from "./components/logs-live-switch";
import { LogsMetricType } from "./components/logs-metric-type";
import { LogsRefresh } from "./components/logs-refresh";
import { LogsSearch } from "./components/logs-search";

export function KeysDetailsLogsControls({
  keyspaceId,
  keyId,
  apiId,
}: {
  keyId: string;
  keyspaceId: string;
  apiId: string;
}) {
  const { data, error, isLoading } = trpc.key.fetchPermissions.useQuery({
    keyId,
    keyspaceId,
  });

  const {
    spentCredits,
    isLoading: spentCreditsLoading,
    isError: spentCreditsError,
  } = useSpentCredits(keyId, keyspaceId);

  const hasRemainingCredit =
    data?.remainingCredit !== null && data?.remainingCredit !== undefined && !isLoading && !error;

  const hasSpentCreditsData = !spentCreditsLoading && !spentCreditsError && spentCredits !== 0;

  // Show credit spent when spent credits data is available (regardless of amount or remaining credits)
  const shouldShowSpentCredits = hasSpentCreditsData && (hasRemainingCredit || spentCredits > 0);

  return (
    <ControlsContainer>
      <ControlsLeft>
        <LogsSearch apiId={apiId} />
        <LogsFilters />
        <LogsMetricType />
        <LogsDateTime />
        <AnimatePresence>
          {hasRemainingCredit ? (
            <motion.div
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
                <motion.div
                  className="text-gray-12 font-medium text-[13px] max-md:hidden pl-4"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: 0.05, duration: 0.2 }}
                >
                  Remaining Credits:
                </motion.div>
                {(data?.remainingCredit ?? 0) > 0 ? (
                  <motion.div
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
                      text={formatRawNumber(data?.remainingCredit ?? 0)}
                      icon={<Coins size="sm-thin" />}
                    />
                  </motion.div>
                ) : (
                  <motion.div
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
                      icon={<Coins size="sm-thin" />}
                    />
                  </motion.div>
                )}
              </div>
            </motion.div>
          ) : null}
        </AnimatePresence>
        <AnimatePresence>
          {shouldShowSpentCredits ? (
            <motion.div
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
                <motion.div
                  className="text-gray-12 font-medium text-[13px] max-md:hidden pl-4"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: 0.05, duration: 0.2 }}
                >
                  Credits Spent:
                </motion.div>
                {spentCredits > 0 ? (
                  <motion.div
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
                      text={formatRawNumber(spentCredits)}
                      icon={<ChartUsage size="sm-thin" className="h-[12px] w-[12px]" />}
                    />
                  </motion.div>
                ) : (
                  <motion.div
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
                      icon={<ChartUsage size="sm-thin" className="h-[12px] w-[12px]" />}
                    />
                  </motion.div>
                )}
              </div>
            </motion.div>
          ) : null}
        </AnimatePresence>
      </ControlsLeft>
      <ControlsRight>
        <LogsLiveSwitch />
        <LogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
