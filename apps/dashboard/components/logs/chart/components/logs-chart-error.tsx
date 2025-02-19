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
        <div className="h-full w-full flex items-center justify-center">
          <span className="text-xs text-error-11 font-mono">Could not retrieve logs</span>
        </div>
      </ResponsiveContainer>
    </div>
  );
};
