import { CircleCarretRight } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useLogsContext } from "../../../context/logs";
import { useFilters } from "../../../hooks/use-filters";

export const LogsLiveSwitch = () => {
  const { isLive, toggleLive } = useLogsContext();
  const { updateFilters } = useFilters();

  const handleSwitch = () => {
    toggleLive();
    // To able to refetch historic data again we have to update the endTime
    if (isLive) {
      const timestamp = Date.now();
      updateFilters([
        {
          field: "endTime",
          value: timestamp,
          id: crypto.randomUUID(),
          operator: "is",
        },
        {
          field: "startTime",
          value: timestamp - 24 * 60 * 60 * 1000,
          id: crypto.randomUUID(),
          operator: "is",
        },
      ]);
    }
  };
  return (
    <Button
      onClick={handleSwitch}
      variant="ghost"
      className={cn(
        "px-2 relative",
        isLive
          ? "bg-info-3 text-info-11 hover:bg-info-3 hover:text-info-11"
          : "text-accent-12 [&_svg]:text-accent-9"
      )}
    >
      {isLive && (
        <div className="absolute left-0 right-0 top-0 bottom-0 rounded">
          <div className="absolute inset-0 bg-info-6 rounded opacity-15 animate-[ping_3s_cubic-bezier(0,0,0.2,1)_infinite]" />
        </div>
      )}
      <CircleCarretRight className="size-4 relative z-10" />
      <span className="font-medium text-[13px]">Live</span>
    </Button>
  );
};
