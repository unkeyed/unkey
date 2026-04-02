"use client";

import { cn } from "@/lib/utils";
import { formatCompoundDuration } from "@/lib/utils/metric-formatters";
import { Check, CircleHalfDottedClock, TriangleWarning2 } from "@unkey/icons";
import { match } from "@unkey/match";
import { Badge, Loading, SettingCard } from "@unkey/ui";
import { GlowIcon } from "../../../../components/glow-icon";

type DeploymentStepProps = {
  icon: React.ReactNode;
  title: string;
  description: string;
  duration?: number;
  status: "pending" | "started" | "completed" | "error" | "skipped";
  statusIcon?: React.ReactNode;
  expandable?: React.ReactNode;
  defaultExpanded?: boolean;
};

export function DeploymentStep({
  icon,
  title,
  description,
  duration,
  status,
  statusIcon,
  expandable,
  defaultExpanded,
}: DeploymentStepProps) {
  const showGlow = status === "started" || status === "error";
  const isError = status === "error";
  const isSkipped = status === "skipped";
  return (
    <div className={cn(isSkipped && "opacity-50")}>
      <SettingCard
        truncateDescription
        icon={
          <GlowIcon
            icon={icon}
            variant={isError ? "error" : "feature"}
            glow={showGlow}
            transition
            className="w-full h-full"
          />
        }
        title={
          <div className="flex items-center gap-2">
            <span>{title}</span>
            {isError ? (
              <Badge
                variant="error"
                size="sm"
                className="transition-all duration-300 font-normal text-[11px] rounded-md h-[18px] opacity-100 scale-100"
              >
                Failed
              </Badge>
            ) : (
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
            )}
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
          <span className="text-gray-10 text-xs">
            {duration !== null && duration !== undefined ? formatCompoundDuration(duration) : null}
          </span>
          {statusIcon ??
            match(status)
              .with("completed", () => (
                <Check iconSize="md-regular" className="text-success-11" />
              ))
              .with("started", () => <Loading className="size-4" />)
              .with("error", () => (
                <TriangleWarning2 className="text-error-11" iconSize="md-regular" />
              ))
              .with("pending", () => (
                <CircleHalfDottedClock className="text-gray-9" iconSize="md-regular" />
              ))
              .with("skipped", () => null)
              .exhaustive()}
        </div>
      </SettingCard>
    </div>
  );
}
