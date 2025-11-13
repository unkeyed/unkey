import { cn } from "@unkey/ui/src/lib/utils";
import { STATUS_CONFIG, type HealthStatus } from "./status-config";

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
    <div className="z-10 mx-auto w-[282px] -m-[20px]">
      <div
        className={cn(
          "h-12 border rounded-t-[14px]",
          config.colors.bannerBg,
          config.colors.bannerBorder
        )}
      >
        <div className="py-1.5 px-2.5 flex items-center">
          <Icon
            className={cn(config.colors.textColor, "shrink-0 mr-2 mb-0.5")}
            iconSize="md-regular"
          />
          <span
            className={cn("text-xs font-medium mr-4", config.colors.textColor)}
          >
            {config.label}
          </span>
          <div className="flex-1 overflow-hidden relative max-w-[200px]">
            <div
              className={cn(
                "text-xs",
                config.colors.textColor,
                "animate-marquee whitespace-nowrap"
              )}
            >
              {config.message}
            </div>
            <div
              className={cn(
                "absolute left-0 top-0 bottom-0 w-6 pointer-events-none",
                config.colors.gradientFromLeft
              )}
            />
            <div
              className={cn(
                "absolute right-0 top-0 bottom-0 w-6 pointer-events-none",
                config.colors.gradientFromRight
              )}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
