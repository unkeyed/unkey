import { Cloud } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";

export function StatusIndicator({
  withSignal = false,
}: {
  withSignal?: boolean;
}) {
  return (
    <div className="relative">
      <div className="size-5 rounded flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100 bg-grayA-3">
        <Cloud iconsize="sm-regular" className="text-gray-12" />
      </div>
      {withSignal && (
        <div className="absolute -top-0.5 -right-0.5">
          {[0, 0.15, 0.3, 0.45].map((delay, index) => (
            <div
              // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
              key={index}
              className={cn(
                "absolute inset-0 size-2 rounded-full",
                index === 0 && "bg-successA-9 opacity-75",
                index === 1 && "bg-successA-10 opacity-60",
                index === 2 && "bg-successA-11 opacity-40",
                index === 3 && "bg-successA-12 opacity-25"
              )}
              style={{
                animation: "ping 2s cubic-bezier(0, 0, 0.2, 1) infinite",
                animationDelay: `${delay}s`,
              }}
            />
          ))}
          <div className="relative size-2 bg-successA-9 rounded-full" />
        </div>
      )}
    </div>
  );
}
