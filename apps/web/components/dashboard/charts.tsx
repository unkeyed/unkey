"use client";

import { Area, Bar, Column } from "@ant-design/plots";
import { useTheme } from "next-themes";

export const useColors = () => {
  const { resolvedTheme } = useTheme();
  return {
    color: resolvedTheme === "dark" ? "#f1efef" : "#1c1917",
    palette:
      resolvedTheme === "dark"
        ? ["#f1efef", "#FFE41C", "#FF7568"]
        : ["#1c1917", "#FFCD07", "#D12542"],
    axisColor: resolvedTheme === "dark" ? "#1b1918" : "#e8e5e3",
  };
};

export type Props = {
  data: {
    x: string;
    y: number;
  }[];
  timeGranularity: "hour" | "day" | "month";
  tooltipLabel: string;
};

export const AreaChart: React.FC<Props> = ({ data, timeGranularity, tooltipLabel }) => {
  const { color, axisColor } = useColors();
  return (
    <Area
      autoFit={true}
      data={data}
      smooth={true}
      // padding={[20, 40, 50, 40]}
      xField="x"
      yField="y"
      color={color}
      areaStyle={{
        fill: `l(270) 0:${color}00  1:${color}`,
      }}
      line={{
        style: {
          lineWidth: 1,
        },
      }}
      xAxis={{
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
          formatter: (v: string) => {
            switch (timeGranularity) {
              case "hour":
                return new Date(v).toLocaleTimeString();
              case "day":
                return new Date(v).toLocaleDateString();
              case "month":
                return new Date(v).toLocaleDateString(undefined, {
                  month: "long",
                  year: "numeric",
                });
            }
          },
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
          name: tooltipLabel,
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
          formatter: (v: string) => new Date(v).toLocaleTimeString(),
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
  padding?: [number, number, number, number];
  data: {
    category: string;
    x: string;
    y: number;
  }[];
  timeGranularity?: "minute" | "hour" | "day" | "month";
}> = ({ data, timeGranularity, padding }) => {
  const { palette, axisColor } = useColors();
  return (
    <Column
      isStack={true}
      color={palette}
      seriesField="category"
      autoFit={true}
      data={data}
      padding={padding ?? [40, 40, 50, 40]}
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
        label: {
          formatter: (v: string) => {
            switch (timeGranularity) {
              case "minute":
                return new Date(v).toLocaleTimeString();
              case "hour":
                return new Date(v).toLocaleTimeString();
              case "day":
                return new Date(v).toLocaleDateString();
              case "month":
                return new Date(v).toLocaleDateString(undefined, {
                  month: "long",
                  year: "numeric",
                });
              default:
                return v;
            }
          },
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

export const StackedBarChart: React.FC<{
  data: {
    category: string;
    x: number;
    y: string;
  }[];
}> = ({ data }) => {
  const { palette, axisColor } = useColors();
  return (
    <Bar
      isStack={true}
      color={palette}
      seriesField="category"
      autoFit={true}
      data={data}
      padding={[40, 50, 40, 120]}
      xField="x"
      yField="y"
      legend={{
        position: "top-right",
      }}
      label={{
        formatter: (d) =>
          d.x > 0 ? Intl.NumberFormat(undefined, { notation: "compact" }).format(d.x) : "",
      }}
      yAxis={{
        label: {
          formatter: (v: string) => {
            return v.length <= 16 ? v : `${v.slice(0, 16)}...`;
          },
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
      xAxis={{
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
          value: Intl.NumberFormat(undefined, { notation: "compact" }).format(Number(datum.x)),
        }),
      }}
    />
  );
};

type ColorCode = "primary" | "warn" | "error";

export const ColumnSpark: React.FC<{
  data: {
    x: string;
    y: number;
    category: string;
    colorCode?: ColorCode;
  }[];
}> = ({ data }) => {
  const { palette } = useColors();

  return (
    <Column
      className="absolute inset-0"
      isStack={true}
      color={palette.slice(0, 2).reverse()}
      seriesField="category"
      autoFit={true}
      data={data}
      padding={[0, 0, 0, 0]}
      xField="x"
      yField="y"
      xAxis={{
        ticks: [],
        line: {
          style: {
            opacity: 0,
          },
        },
        grid: {
          line: {
            style: { opacity: 0 },
          },
        },
      }}
      yAxis={{
        ticks: [],

        line: {
          style: {
            opacity: 0,
          },
        },
        grid: {
          line: {
            style: { opacity: 0 },
          },
        },
      }}
      legend={false}
      tooltip={{
        formatter: (datum) => ({
          name: datum.category,
          value: Intl.NumberFormat(undefined, { notation: "compact" }).format(Number(datum.y)),
        }),
      }}
    />
  );
};
