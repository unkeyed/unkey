import { ChartError as BaseChartError } from "@/components/logs/chart/chart-states";

type ChartErrorProps = {
  height: number;
};

export function ChartError({ height }: ChartErrorProps) {
  return (
    <div className="w-full flex items-center justify-center" style={{ height }}>
      <BaseChartError variant="simple" className="**:text-xs" />
    </div>
  );
}
