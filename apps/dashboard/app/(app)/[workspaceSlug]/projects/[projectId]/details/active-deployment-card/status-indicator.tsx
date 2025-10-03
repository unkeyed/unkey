import { Cloud } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";

export type DiffStatus = "breaking" | "warning" | "safe" | "loading";

type StatusIndicatorProps = {
  status?: DiffStatus;
  withSignal?: boolean;
  className?: string;
};

const getTooltipContent = (status: DiffStatus): string => {
  switch (status) {
    case "breaking":
      return "Breaking changes detected - this deployment may break existing API clients";
    case "warning":
      return "API changes detected - review the differences before deploying";
    case "safe":
      return "No API changes detected";
    case "loading":
      return "Analyzing API differences...";
  }
};

export function StatusIndicator({
  status = "safe",
  withSignal = false,
  className,
}: StatusIndicatorProps) {
  const isBreaking = status === "breaking";
  const isWarning = status === "warning";
  const isLoading = status === "loading";

  const pulseColors = isBreaking
    ? ["bg-error-9", "bg-error-10", "bg-error-11", "bg-error-12"]
    : isWarning
      ? ["bg-warning-9", "bg-warning-10", "bg-warning-11", "bg-warning-12"]
      : ["bg-successA-9", "bg-successA-10", "bg-successA-11", "bg-successA-12"];

  const coreColor = isBreaking ? "bg-error-9" : isWarning ? "bg-warning-9" : "bg-successA-9";

  return (
    <InfoTooltip
      content={getTooltipContent(status)}
      position={{
        side: "bottom",
        align: "center",
      }}
      className="max-w-[300px]"
    >
      <div className="relative">
        <div
          className={cn(
            "size-5 rounded flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100 bg-grayA-3",
            className,
          )}
        >
          <Cloud iconsize="sm-regular" className="text-gray-12" />
        </div>
        {withSignal && !isLoading && (
          <div className="absolute -top-0.5 -right-0.5">
            <PulseIndicator colors={pulseColors} coreColor={coreColor} />
          </div>
        )}
      </div>
    </InfoTooltip>
  );
}

type PulseIndicatorProps = {
  colors?: string[];
  coreColor?: string;
  className?: string;
};

export function PulseIndicator({
  colors = ["bg-successA-9", "bg-successA-10", "bg-successA-11", "bg-successA-12"],
  coreColor = "bg-successA-9",
  className,
}: PulseIndicatorProps) {
  return (
    <div className={cn("relative", className)}>
      {[0, 0.15, 0.3, 0.45].map((delay, index) => (
        <div
          // biome-ignore lint/suspicious/noArrayIndexKey: its okay
          key={index}
          className={cn(
            "absolute inset-0 size-2 rounded-full",
            colors[index],
            index === 0 && "opacity-75",
            index === 1 && "opacity-60",
            index === 2 && "opacity-40",
            index === 3 && "opacity-25",
          )}
          style={{
            animation: "ping 2s cubic-bezier(0, 0, 0.2, 1) infinite",
            animationDelay: `${delay}s`,
          }}
        />
      ))}
      <div className={cn("relative size-2 rounded-full", coreColor)} />
    </div>
  );
}
