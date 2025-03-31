import { ResponsiveContainer } from "recharts";

export const LogsChartError = () => {
  return (
    <div className="w-full relative">
      <div className="px-2 text-accent-11 font-mono absolute top-0 text-xxs w-full flex justify-between opacity-50">
        {Array(5)
          .fill(0)
          .map((_, i) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: it's okay
            <div key={i} className="z-10">
              --:--
            </div>
          ))}
      </div>
      <ResponsiveContainer height={50} className="border-b border-gray-4" width="100%">
        <div className="flex-1 flex items-center justify-center h-full">
          <div className="flex flex-col items-center gap-2">
            <span className="text-sm text-accent-9">Could not retrieve logs</span>
          </div>
        </div>
      </ResponsiveContainer>
    </div>
  );
};
