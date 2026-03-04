"use client";

import { cn } from "@/lib/utils";
import { Check, TriangleWarning2 } from "@unkey/icons";
import { Badge, Loading, SettingCard } from "@unkey/ui";
import ms from "ms";

type DeploymentStepProps = {
  icon: React.ReactNode;
  title: string;
  description: string;
  duration?: number;
  status: "pending" | "started" | "completed" | "error";
  expandable?: React.ReactNode;
  defaultExpanded?: boolean;
  badgeText?: string;
};

export function DeploymentStep({
  icon,
  title,
  description,
  duration,
  status,
  expandable,
  defaultExpanded,
}: DeploymentStepProps) {
  const showGlow = status === "started";
  return (
    <SettingCard
      icon={
        <div className="relative w-full h-full">
          <div
            className={cn(
              "absolute inset-[-4px] rounded-[10px] blur-[14px]",
              "bg-linear-to-l from-feature-8 to-info-9",
              showGlow ? "animate-pulse opacity-20" : "opacity-0 transition-opacity duration-300",
            )}
          />
          <div
            className={cn(
              "w-full h-full rounded-[10px] flex items-center justify-center shrink-0",
              showGlow &&
                "relative dark:bg-white dark:text-black bg-black text-white shadow-md shadow-black/40",
            )}
          >
            {icon}
          </div>
        </div>
      }
      title={
        <div className="flex items-center gap-2">
          <span>{title}</span>
          <Badge
            variant="success"
            size="sm"
            className={cn(
              "transition-all duration-300 font-normal text-[11px] rounded-md h-[18px]",
              status === "completed" ? "opacity-100 scale-100" : "opacity-0 scale-95",
            )}
          >
            Complete
          </Badge>
        </div>
      }
      iconClassName={showGlow ? "bg-transparent shadow-none dark:ring-0" : undefined}
      className="relative"
      description={description}
      expandable={expandable}
      defaultExpanded={defaultExpanded}
      contentWidth="w-fit"
    >
      <div className="flex items-center gap-4 justify-end w-full absolute right-14">
        <span className="text-gray-10 text-xs">{duration ? ms(duration) : null}</span>
        {status === "completed" ? (
          <Check iconSize="md-regular" className="text-success-11" />
        ) : status === "started" ? (
          <Loading className="size-4" />
        ) : status === "error" ? (
          <TriangleWarning2 className="text-error-11" />
        ) : null}
      </div>
    </SettingCard>
  );
}
