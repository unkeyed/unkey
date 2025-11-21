import { InfoTooltip } from "@unkey/ui";
import type { HealthStatus } from "../types";
import { STATUS_CONFIG } from "./status-config";
import { StatusDot } from "./status-dot";

export type StatusIndicatorOrientation = "vertical" | "horizontal";
type StatusIndicatorProps = {
  icon: React.ReactNode;
  healthStatus: HealthStatus;
  tooltip: string;
  showGlow?: boolean;
  orientation?: StatusIndicatorOrientation;
};

export function StatusIndicator({
  icon,
  healthStatus,
  tooltip,
  showGlow = false,
  orientation = "vertical",
}: StatusIndicatorProps) {
  const { colors } = STATUS_CONFIG[healthStatus];
  const glowBoxShadow = showGlow
    ? `0 0 8px 1px ${colors.dotRing} inset, 0 0 0 1px var(--color-grayA-gray-a5, rgba(0, 9, 50, 0.12)) inset`
    : "";

  if (orientation === "horizontal") {
    return (
      <InfoTooltip
        content={tooltip}
        variant="primary"
        asChild
        className="px-2.5 py-1 rounded-[10px] text-whiteA-12 bg-blackA-12 text-xs z-30"
        position={{ align: "center", side: "top", sideOffset: 5 }}
      >
        <div
          className="border bg-gray-1 border-grayA-3 h-[22px] w-14 rounded-full flex transition-all hover:ring-1 hover:ring-gray-7 duration-200 ease-out hover:scale-105 cursor-pointer overflow-hidden flex-row-reverse"
          style={{
            boxShadow: glowBoxShadow,
          }}
        >
          <div className="w-1/2 border-r border-grayA-3 relative flex items-center justify-center flex-shrink-0">
            <StatusDot healthStatus={healthStatus} />
          </div>
          <div className="w-1/2 bg-grayA-2 flex items-center justify-center flex-shrink-0">
            {icon}
          </div>
        </div>
      </InfoTooltip>
    );
  }

  return (
    <InfoTooltip
      content={tooltip}
      variant="primary"
      asChild
      className="px-2.5 py-1 rounded-[10px] text-whiteA-12 bg-blackA-12 text-xs z-30"
      position={{ align: "center", side: "top", sideOffset: 5 }}
    >
      <div
        className="border bg-gray-1 border-grayA-3 h-full rounded-lg w-8 transition-all hover:ring-1 hover:ring-gray-7 duration-200 ease-out hover:scale-105 cursor-pointer overflow-hidden"
        style={{
          boxShadow: glowBoxShadow,
        }}
      >
        <div className="h-6 border-b border-grayA-3 relative">
          <StatusDot healthStatus={healthStatus} />
        </div>
        <div className="h-5 bg-grayA-2 pl-1 pt-[3px]">{icon}</div>
      </div>
    </InfoTooltip>
  );
}
