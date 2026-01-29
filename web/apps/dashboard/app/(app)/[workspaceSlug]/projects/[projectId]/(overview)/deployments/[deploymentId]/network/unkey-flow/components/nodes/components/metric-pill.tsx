import { InfoTooltip } from "@unkey/ui";

type MetricPillProps = {
  icon: React.ReactNode;
  value: string | number;
  tooltip: string;
};

export function MetricPill({ icon, value, tooltip }: MetricPillProps) {
  return (
    <InfoTooltip
      content={tooltip}
      variant="primary"
      className="px-2.5 py-1 rounded-[10px] text-whiteA-12 bg-blackA-12 text-xs z-30"
      position={{ align: "center", side: "top", sideOffset: 5 }}
    >
      <div className="bg-grayA-3 p-1.5 flex items-center justify-between rounded-full h-5 gap-1.5 transition-all hover:bg-grayA-4 cursor-pointer">
        {icon}
        <span className="text-gray-9 text-[10px] tabular-nums">{value}</span>
      </div>
    </InfoTooltip>
  );
}
