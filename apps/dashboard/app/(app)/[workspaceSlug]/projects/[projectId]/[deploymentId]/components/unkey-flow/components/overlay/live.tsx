import { InfoTooltip } from "@unkey/ui";
import { StatusDot } from "../nodes/status-dot";

export const LiveIndicator = () => {
  return (
    <div className="absolute top-4 right-4 px-2 py-1.5 dark:bg-black bg-white rounded-full ring-1 ring-grayA-5 flex items-center justify-between gap-2 h-6 pointer-events-auto">
      <InfoTooltip
        content="Live monitoring enabled. Metrics refresh every 10s"
        variant="primary"
        className="px-2.5 py-1 rounded-[10px] bg-blackA-12 text-xs z-30 text-white"
        position={{ align: "center", side: "top", sideOffset: 5 }}
      >
        <div className="bg-base-12 flex items-center justify-between gap-2 cursor-pointer">
          <StatusDot healthStatus="health_syncing" variant="relative" />
          <span className="text-accent-12 font-medium text-[13px]">Live</span>
        </div>
      </InfoTooltip>
    </div>
  );
};
