import { ResponsiveContainer } from "recharts";

export const LogsChartError = () => {
  return (
    <div className="w-full relative">
      <ResponsiveContainer height={50} className="border-b border-grayA-4" width="100%">
        <div className="flex-1 flex items-center justify-center h-full">
          <div className="flex flex-col items-center gap-2">
            <span className="text-sm text-accent-9">Could not retrieve logs</span>
          </div>
        </div>
      </ResponsiveContainer>
    </div>
  );
};
