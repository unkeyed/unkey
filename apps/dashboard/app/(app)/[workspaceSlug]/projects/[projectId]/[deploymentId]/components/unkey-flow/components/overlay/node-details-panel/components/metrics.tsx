import type { ChartConfig } from "@/components/ui/chart";
import { ChevronExpandY } from "@unkey/icons";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { generateRealisticChartData } from "../utils";
import { LogsTimeseriesBarChart } from "./chart";

const baseConfig = {
  startTime: Date.now() - 24 * 60 * 60 * 1000 * 5,
  intervalMs: 60 * 60 * 1000,
};

export const Metrics = ({
  metrics,
}: {
  metrics: {
    icon: React.ReactNode;
    label: string;
    value: React.ReactNode;
    config: ChartConfig;
    dataConfig: Parameters<typeof generateRealisticChartData>[0];
  }[];
}) => {
  return (
    <div>
      <div className="flex px-4 w-full">
        <div className="flex items-center gap-3 w-full">
          <div className="text-gray-9 text-xs whitespace-nowrap">Runtime metrics</div>
          <div className="h-0.5 bg-grayA-3 rounded-sm flex-1 min-w-[115px]" />
          <div className="flex items-center gap-2 shrink-0">
            <Select>
              <SelectTrigger
                className="rounded-lg !px-2 !py-1.5 text-gray-10 text-xs !min-h-[26px]"
                rightIcon={<ChevronExpandY className="ml-2 text-gray-10" />}
              >
                <SelectValue placeholder="24H" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="24h">24H</SelectItem>
                <SelectItem value="7d">7D</SelectItem>
                <SelectItem value="30d">30D</SelectItem>
              </SelectContent>
            </Select>
            <Select>
              <SelectTrigger
                className="rounded-lg !px-2 !py-1.5 text-gray-10 text-xs !min-h-[26px]"
                rightIcon={<ChevronExpandY className="ml-2 text-gray-10" />}
              >
                <SelectValue placeholder="PST" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="pst">PST</SelectItem>
                <SelectItem value="est">EST</SelectItem>
                <SelectItem value="utc">UTC</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>
      {metrics.map((metric, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
        <div key={index} className="flex flex-col gap-3 px-4 w-full mt-5">
          <div className="flex gap-3 items-center">
            <div className="bg-grayA-3 text-gray-12 rounded-md size-[22px] items-center flex justify-center">
              {metric.icon}
            </div>
            <span className="text-gray-11 text-xs">{metric.label}</span>
            <div className="ml-10">{metric.value}</div>
          </div>
          <LogsTimeseriesBarChart
            data={generateRealisticChartData({
              ...metric.dataConfig,
              ...baseConfig,
            })}
            config={metric.config}
            height={48}
            isLoading={false}
            isError={false}
          />
        </div>
      ))}
    </div>
  );
};
