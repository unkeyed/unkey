"use client";

import { Area, Column } from "@ant-design/plots";
import { useTheme } from "next-themes";

const useColors = () => {
  const { resolvedTheme } = useTheme();
  return {
    color: resolvedTheme === "dark" ? "#f1efef" : "#1c1917",
    palette:
      resolvedTheme === "dark"
        ? ["#f1efef", "#FFE41C", "#FF7568"]
        : ["#1c1917", "#BA7901", "#A01649"],
    axisColor: resolvedTheme === "dark" ? "#1b1918" : "#e8e5e3",
  };
};

type Props = {
  data: {
    x: string;
    y: number;
  }[];
};

export const AreaChart: React.FC<Props> = ({ data }) => {
  const { color, axisColor } = useColors();
  return (
    <Area
      autoFit={true}
      data={data}
      smooth={true}
      padding={[20, 40, 50, 40]}
      xField="x"
      yField="y"
      color={color}
      xAxis={{
        tickCount: 3,
        tickLine: {
          style: {
            stroke: axisColor,
          },
        },
        line: {
          style: {
            stroke: axisColor,
          },
        },
        label: {
          formatter: (v: string) => new Date(v).toLocaleDateString(),
        },
      }}
      yAxis={{
        tickCount: 3,
        label: {
          formatter: (v: string) =>
            Intl.NumberFormat(undefined, { notation: "compact" }).format(Number(v)),
        },
        grid: {
          line: {
            style: {
              stroke: axisColor,
            },
          },
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
  const { color, axisColor } = useColors();
  return (
    <Column
      color={color}
      autoFit={true}
      data={data}
      padding={[20, 40, 50, 40]}
      xField="x"
      yField="y"
      xAxis={{
        maxTickCount: 5,
        label: {
          formatter: (v: string) => new Date(v).toLocaleDateString(),
        },
        tickLine: {
          style: {
            stroke: axisColor,
          },
        },
        line: {
          style: {
            stroke: axisColor,
          },
        },
      }}
      yAxis={{
        tickCount: 5,
        label: {
          formatter: (v: string) =>
            Intl.NumberFormat(undefined, { notation: "compact" }).format(Number(v)),
        },
        grid: {
          line: {
            style: {
              stroke: axisColor,
            },
          },
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

export const StackedColumnChart: React.FC<{
  data: {
    category: string;
    x: string;
    y: number;
  }[];
}> = ({ data }) => {
  const { palette, axisColor } = useColors();
  return (
    <Column
      isStack={true}
      color={palette}
      seriesField="category"
      autoFit={true}
      data={data}
      padding={[40, 40, 50, 40]}
      xField="x"
      yField="y"
      legend={{
        position: "top-right",
      }}
      interactions={[
        {
          type: "active-region",
          enable: false,
        },
      ]}
      connectedArea={{
        style: (oldStyle, _element) => {
          return {
            fill: "rgba(0,0,0,0.25)",
            stroke: oldStyle.fill,
            lineWidth: 0.5,
          };
        },
      }}
      xAxis={{
        maxTickCount: 5,
        label: {
          formatter: (v: string) => new Date(v).toLocaleDateString(),
        },
        tickLine: {
          style: {
            stroke: axisColor,
          },
        },
        line: {
          style: {
            stroke: axisColor,
          },
        },
      }}
      yAxis={{
        tickCount: 5,
        label: {
          formatter: (v: string) =>
            Intl.NumberFormat(undefined, { notation: "compact" }).format(Number(v)),
        },
        grid: {
          line: {
            style: {
              stroke: axisColor,
            },
          },
        },
      }}
      tooltip={{
        formatter: (datum) => ({
          name: datum.category,
          value: Intl.NumberFormat(undefined, { notation: "compact" }).format(Number(datum.y)),
        }),
      }}
    />
  );
};
