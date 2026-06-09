import { cn } from "@unkey/ui/src/lib/utils";
import { type HealthStatus, STATUS_CONFIG } from "../status/status-config";
import { DEFAULT_NODE_WIDTH } from "../types";

type HealthBannerProps = {
  healthStatus: HealthStatus;
  className?: string;
};

export function HealthBanner({ healthStatus }: HealthBannerProps) {
  const config = STATUS_CONFIG[healthStatus];

  if (!config.showBanner) {
    return null;
  }

  const Icon = config.icon;

  return (
    <div className={`mx-auto w-[${DEFAULT_NODE_WIDTH}px] -m-[20px]`}>
      <div
        className={cn(
          "h-12 border rounded-t-[14px]",
          config.colors.bannerBg,
          config.colors.bannerBorder,
        )}
      >
        <div className="py-1.5 px-2.5 flex items-center">
          <Icon
            className={cn(config.colors.textColor, "shrink-0 mr-2 mb-0.5")}
            iconSize="md-regular"
          />
          <span className={cn("text-xs font-medium mr-4", config.colors.textColor)}>
            {config.label}
          </span>
          <div className="flex-1 overflow-hidden max-w-[200px] grid">
            <div
              className={cn(
                "[grid-area:1/1] text-xs animate-marquee whitespace-nowrap",
                config.colors.textColor,
              )}
            >
              {config.message}
            </div>
            <div className="[grid-area:1/1] flex justify-between pointer-events-none">
              <div className={cn("w-6 h-full", config.colors.gradientFromLeft)} />
              <div className={cn("w-6 h-full ml-auto", config.colors.gradientFromRight)} />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
