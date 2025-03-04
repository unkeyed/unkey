export const LogsChartError = () => {
  return (
    <div className="flex flex-col h-full">
      <div className="flex-1 min-h-0 flex items-center justify-center">
        <div className="flex flex-col items-center gap-2">
          <span className="text-sm text-accent-9">Could not retrieve logs</span>
        </div>
      </div>
    </div>
  );
};
