import { StatusBadge } from "@/app/(app)/apis/[apiId]/settings/components/status-badge";
import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { formatNumber } from "@/lib/fmt";
import { ChartUsage } from "@unkey/icons";
import { Separator } from "@unkey/ui";
import { AnimatePresence, motion } from "framer-motion";
import { useApiSpentCredits } from "../../hooks/use-spent-credits";
import { LogsDateTime } from "./components/logs-datetime";
import { LogsFilters } from "./components/logs-filters";
import { LogsRefresh } from "./components/logs-refresh";
import { LogsSearch } from "./components/logs-search";

export function KeysOverviewLogsControls({ apiId }: { apiId: string }) {
  const {
    spentCredits,
    isLoading: spentCreditsLoading,
    isError: spentCreditsError,
  } = useApiSpentCredits(apiId);

  const hasSpentCreditsData = !spentCreditsLoading && !spentCreditsError;

  return (
    <ControlsContainer>
      <ControlsLeft>
        <LogsSearch apiId={apiId} />
        <LogsFilters />
        <LogsDateTime />
        <AnimatePresence>
          {hasSpentCreditsData && spentCredits > 0 ? (
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
                    text={formatNumber(spentCredits)}
                    icon={<ChartUsage size="sm-thin" />}
                  />
                </motion.div>
              </div>
            </motion.div>
          ) : null}
        </AnimatePresence>
      </ControlsLeft>
      <ControlsRight>
        <LogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
