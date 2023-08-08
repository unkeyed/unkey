"use client";

import { Area, Column } from "@ant-design/plots";
import { useTheme } from "next-themes";
type Props = {
  data: {
    x: string;
    y: number;
  }[];
};

export const AreaChart: React.FC<Props> = ({ data }) => {
  const { theme } = useTheme();
  return (
    <Area
      autoFit={true}
      data={data}
      smooth={true}
      padding={[40, 40, 30, 40]}
      xField="x"
      yField="y"
      theme={theme}
      xAxis={{
        tickCount: 3,
      }}
      yAxis={{
        tickCount: 3,
        label: {
          formatter: (v: string) =>
            Intl.NumberFormat(undefined, { notation: "compact" }).format(Number(v)),
        },
      }}
      tooltip={{
        formatter: (datum) => ({
          name: "Events",
          value: Intl.NumberFormat(undefined, { notation: "compact" }).format(Number(datum.y)),
        }),
      }}
    />
  );
};

export const ColumnChart: React.FC<Props> = ({ data }) => {
  const { theme } = useTheme();
  return (
    <Column
      theme={theme}
      autoFit={true}
      data={data}
      padding={[40, 40, 30, 40]}
      xField="x"
      yField="y"
      yAxis={{
        tickCount: 3,
        label: {
          formatter: (v: string) =>
            Intl.NumberFormat(undefined, { notation: "compact" }).format(Number(v)),
        },
      }}
      tooltip={{
        formatter: (datum) => ({
          name: "Usage",
          value: Intl.NumberFormat(undefined, { notation: "compact" }).format(Number(datum.y)),
        }),
      }}
    />
  );
};
