type ChartErrorProps = {
  height: number;
  message?: string;
};

export function ChartError({ height, message = "Could not retrieve data" }: ChartErrorProps) {
  return (
    <div className="w-full flex items-center justify-center" style={{ height }}>
      <span className="text-xs text-accent-9">{message}</span>
    </div>
  );
}
