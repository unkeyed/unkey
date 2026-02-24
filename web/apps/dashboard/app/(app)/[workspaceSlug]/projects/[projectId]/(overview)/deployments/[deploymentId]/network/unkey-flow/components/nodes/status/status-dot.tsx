import { cn } from "@unkey/ui/src/lib/utils";
import { type HealthStatus, STATUS_CONFIG } from "./status-config";

type StatusDotProps = {
  healthStatus: HealthStatus;
};

export function StatusDot({ healthStatus }: StatusDotProps) {
  const { colors } = STATUS_CONFIG[healthStatus];

  return (
    <>
      <div className="size-[7px] grid">
        {/* Ring 1 */}
        <div
          className="[grid-area:1/1] rounded-full breathe-ring"
          style={{
            animation: "breathe-ring 2s ease-in-out infinite",
            boxShadow: `0 0 0 1.5px ${colors.dotRing}`,
          }}
        />
        {/* Ring 2 */}
        <div
          className="[grid-area:1/1] rounded-full breathe-ring"
          style={{
            animation: "breathe-ring 2s ease-in-out infinite 1s",
            boxShadow: `0 0 0 1.5px ${colors.dotRing}`,
          }}
        />
        {/* Solid dot */}
        <div className={cn("[grid-area:1/1] rounded-full", colors.dotBg)} />
      </div>
      <style>{`
        @keyframes breathe-ring {
          0%, 100% {
            transform: scale(1);
            opacity: 0.6;
          }
          50% {
            transform: scale(2.2);
            opacity: 0;
          }
        }
      `}</style>
    </>
  );
}
