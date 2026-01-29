import { Heart } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import { type HealthStatus, STATUS_CONFIG } from "../status/status-config";
import { StatusIndicator } from "../status/status-indicator";

type CardHeaderVariant = "card" | "panel";

export type CardHeaderProps = {
  icon: React.ReactNode;
  title: string;
  subtitle: string;
  health: HealthStatus;
  variant?: CardHeaderVariant;
};

export function CardHeader({ icon, title, subtitle, health, variant = "card" }: CardHeaderProps) {
  const { colors } = STATUS_CONFIG[health];
  const isCard = variant === "card";

  return (
    <div
      className={cn(
        "flex w-full",
        isCard && "border-b border-grayA-4 rounded-t-[14px] px-3 py-2.5 ",
      )}
      style={
        isCard
          ? {
              background:
                "radial-gradient(circle at 5% 15%, hsl(var(--grayA-3)) 0%, transparent 20%), light-dark(#FFF, #000)",
            }
          : undefined
      }
    >
      <div className="flex items-center justify-between gap-3">
        {icon}
        <div className="flex flex-col gap-[3px] justify-center h-9 py-2">
          <div className="text-accent-12 font-medium text-xs font-mono">{title}</div>
          <div className="text-gray-9 text-[11px]">{subtitle}</div>
        </div>
      </div>
      <div className="flex gap-2 items-center ml-auto">
        <StatusIndicator
          icon={<Heart className={colors.dotTextColor} iconSize="sm-regular" />}
          healthStatus={health}
          tooltip="Sentinel health status"
          showGlow={health !== "normal"}
        />
      </div>
    </div>
  );
}
