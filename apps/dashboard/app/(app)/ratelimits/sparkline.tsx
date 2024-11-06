"use client";
import { AreaChart } from "@/lib/charts/area-chart";
import { SparkLine as Chart } from "@/lib/charts/sparkline";
import { nFormatter } from "@/lib/format";
type Props = {
  data: {
    time: number;
    values: {
      passed: number;
      total: number;
    };
  }[];
};

export const Sparkline: React.FC<Props> = ({ data }) => {
  const data2 = data.map((d) => ({
    date: new Date(d.time),
    values: d.values,
  }));
  return (
    <div className="w-full h-full">
      <Chart
        key={"xx"}
        data={data2}
        series={[
          { id: "total", valueAccessor: (d) => d.values.total, color: "text-warn" },
          { id: "passed", valueAccessor: (d) => d.values.passed, color: "text-primary" },
        ]}
        tooltipContent={(d) => (
          <>
            <p className="text-content-subtle">
              <strong className="text-content">
                {nFormatter(d.values.passed, { full: true })}
              </strong>{" "}
              /{" "}
              <strong className="text-content">{nFormatter(d.values.total, { full: true })}</strong>{" "}
              Passed
            </p>
            <p className="text-sm text-content-subtle">{new Date(d.date).toLocaleTimeString()}</p>
          </>
        )}
      >
        <AreaChart />
      </Chart>
    </div>
  );
};
