import { InfoTooltip } from "@unkey/ui";
import { StatusDot } from "../nodes/status/status-dot";

export const LiveIndicator = () => {
  return (
    <div className="absolute top-4 right-4 px-2 py-1.5 bg-raised rounded-full border flex items-center justify-between gap-2 h-6 pointer-events-auto shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)]">
      <InfoTooltip
        content="Live monitoring enabled. Metrics refresh every 10s"
        className="z-30"
        position={{ align: "center", side: "top", sideOffset: 5 }}
      >
        <div className="bg-raised flex items-center justify-between gap-2 cursor-pointer">
          <StatusDot healthStatus="health_syncing" />
          <span className="text-gray-12 font-medium text-[13px]">Live</span>
        </div>
      </InfoTooltip>
    </div>
  );
};
