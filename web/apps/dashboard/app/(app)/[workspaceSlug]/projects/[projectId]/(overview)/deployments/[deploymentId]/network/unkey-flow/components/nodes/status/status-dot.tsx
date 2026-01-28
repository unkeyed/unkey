import { cn } from "@unkey/ui/src/lib/utils";
import { type HealthStatus, STATUS_CONFIG } from "./status-config";

type StatusDotProps = {
  healthStatus: HealthStatus;
  variant?: "absolute" | "relative";
};

export function StatusDot({ healthStatus, variant = "absolute" }: StatusDotProps) {
  const { colors } = STATUS_CONFIG[healthStatus];

  const wrapperClass =
    variant === "absolute" ? "absolute top-1.5 right-1.5 size-[7px]" : "relative size-[7px]";

  return (
    <>
      <div className={wrapperClass}>
        {/* Ring 1 */}
        <div
          className="absolute inset-0 rounded-full"
          style={{
            animation: "breathe-ring 2s ease-in-out infinite",
            boxShadow: `0 0 0 1.5px ${colors.dotRing}`,
          }}
        />
        {/* Ring 2 */}
        <div
          className="absolute inset-0 rounded-full"
          style={{
            animation: "breathe-ring 2s ease-in-out infinite 1s",
            boxShadow: `0 0 0 1.5px ${colors.dotRing}`,
          }}
        />
        {/* Solid dot */}
        <div className={cn("absolute inset-0 rounded-full", colors.dotBg)} />
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
