import { cn } from "@unkey/ui/src/lib/utils";

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
