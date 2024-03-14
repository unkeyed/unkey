"use client";
import { curveMonotoneX } from "@visx/curve";
import { LinearGradient } from "@visx/gradient";
import { Group } from "@visx/group";
import { Area, AreaClosed } from "@visx/shape";
import { motion } from "framer-motion";
import { Fragment, useMemo } from "react";
import { useChartContext } from "./chart-context";

export function AreaChart() {
  const { data, series, margin, xScale, yScale } = useChartContext();

  // Data with all values set to zero to animate from
  const zeroedData = useMemo(() => {
    return data.map((d) => ({
      ...d,
      values: Object.fromEntries(Object.keys(d.values).map((key) => [key, 0])),
    })) as typeof data;
  }, [data]);

  const hasData = useMemo(
    () => data.some((d) => Object.values(d.values).some((v) => v > 0)),
    [data],
  );

  return (
    <Group left={margin.left} top={margin.top}>
      {series.map((s) => (
        <Fragment key={s.id}>
          {/* Area background gradient */}
          <LinearGradient
            className={s.color as string}
            id={`${s.id}-${hasData}-background`}
            fromOffset="20%"
            from="currentColor"
            fromOpacity={hasData ? 0.01 : 0}
            to="currentColor"
            toOpacity={hasData ? 0.2 : 0}
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
                  fill={`url(#${s.id}-${hasData}-background)`}
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
