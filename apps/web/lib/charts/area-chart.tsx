"use client";
import { curveMonotoneX } from "@visx/curve";
import { LinearGradient } from "@visx/gradient";
import { Group } from "@visx/group";
import { Area, AreaClosed, Circle } from "@visx/shape";
import { motion } from "framer-motion";
import { Fragment, useMemo } from "react";
import { useChartContext, useChartTooltipContext } from "./chart-context";

export function AreaChart() {
  const { data, series, margin, xScale, yScale } = useChartContext();
  const { tooltipData } = useChartTooltipContext();

  // Data with all values set to zero to animate from
  const zeroedData = useMemo(() => {
    return data.map((d) => ({
      ...d,
      values: Object.fromEntries(Object.keys(d.values).map((key) => [key, 0])),
    })) as typeof data;
  }, [data]);

  return (
    <Group left={margin.left} top={margin.top}>
      {series.map((s) => (
        <Fragment key={s.id}>
          {/* Area background gradient */}
          <LinearGradient
            className={s.color as string}
            id={`${s.id}-background`}
            fromOffset="20%"
            from="currentColor"
            fromOpacity={0.01}
            to="currentColor"
            toOpacity={0.2}
            x1={0}
            x2={0}
            y1={1}
          />

          {/* Area */}
          <AreaClosed
            data={data}
            x={(d) => xScale(d.date)}
            y={(d) => yScale(s.valueAccessor(d) ?? 0)}
            yScale={yScale}
          >
            {({ path }) => {
              return (
                <motion.path
                  initial={{ d: path(zeroedData) || "" }}
                  animate={{ d: path(data) || "" }}
                  fill={`url(#${s.id}-background)`}
                />
              );
            }}
          </AreaClosed>

          {/* Line */}
          <Area
            curve={curveMonotoneX}
            data={data}
            x={(d) => xScale(d.date)}
            y={(d) => yScale(s.valueAccessor(d) ?? 0)}
          >
            {({ path }) => (
              <motion.path
                initial={{ d: path(zeroedData) || "" }}
                animate={{ d: path(data) || "" }}
                className={s.color as string}
                stroke="currentColor"
                strokeOpacity={0.8}
                strokeWidth={1}
              />
            )}
          </Area>
        </Fragment>
      ))}
    </Group>
  );
}
