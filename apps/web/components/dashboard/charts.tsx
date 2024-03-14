"use client";

import { Area, Bar, Column } from "@ant-design/plots";
import { useTheme } from "next-themes";

type ColorName = "primary" | "warn" | "danger";

export const useColors = (colorNames: Array<ColorName>) => {
  const { resolvedTheme } = useTheme();

  const colors: { light: Record<ColorName, string>; dark: Record<ColorName, string> } = {
    light: {
      primary: "#1c1917",
      warn: "#FFCD07",
      danger: "#D12542",
    },
    dark: {
      primary: "#f1efef",
      warn: "#FFE41C",
      danger: "#FF7568",
    },
  };

  return {
    color: resolvedTheme === "dark" ? "#f1efef" : "#1c1917",
    palette:
      resolvedTheme === "dark"
        ? colorNames.map((c) => colors.dark[c])
        : colorNames.map((c) => colors.light[c]),
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
  colors?: Array<ColorName>;
};

export const AreaChart: React.FC<Props> = ({ data, timeGranularity, tooltipLabel }) => {
  const { color, axisColor } = useColors(["primary", "warn", "danger"]);
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

export const ColumnChart: React.FC<Props> = ({ data, colors }) => {
  const { color, axisColor } = useColors(colors ?? ["primary", "warn", "danger"]);
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
  colors: Array<ColorName>;
}> = ({ data, timeGranularity, padding, colors }) => {
  const { palette, axisColor } = useColors(colors);
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
  colors: Array<ColorName>;
}> = ({ data, colors }) => {
  const { palette, axisColor } = useColors(colors);
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
      maxBarWidth={16}
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
