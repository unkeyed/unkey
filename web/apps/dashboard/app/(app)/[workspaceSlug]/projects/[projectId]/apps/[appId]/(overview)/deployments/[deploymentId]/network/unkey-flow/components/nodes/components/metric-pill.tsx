import { InfoTooltip } from "@unkey/ui";

type MetricPillProps = {
  icon: React.ReactNode;
  value: React.ReactNode;
  tooltip: string;
};

export function MetricPill({ icon, value, tooltip }: MetricPillProps) {
  return (
    <InfoTooltip
      content={tooltip}
      className="z-30"
      position={{ align: "center", side: "top", sideOffset: 5 }}
    >
      <div className="bg-grayA-3 p-1.5 flex items-center justify-between rounded-full h-5 gap-1.5 transition-all hover:bg-grayA-4 cursor-pointer">
        {icon}
        <span className="text-gray-9 text-3xs tabular-nums">{value}</span>
      </div>
    </InfoTooltip>
  );
}
