"use client";
import { Group } from "@visx/group";
import { ParentSize } from "@visx/responsive";
import { scaleLinear, scaleUtc } from "@visx/scale";
import { Bar, Circle, Line } from "@visx/shape";
import { type PropsWithChildren, useMemo, useState } from "react";
import { ChartContext, ChartTooltipContext } from "./chart-context";
import type { ChartContext as ChartContextType, ChartProps, Datum } from "./types";
import { useTooltip } from "./useTooltip";

type SparkLineProps<T extends Datum> = PropsWithChildren<ChartProps<T>>;

export function SparkLine<T extends Datum>(props: SparkLineProps<T>) {
  return (
    <ParentSize className="relative">
      {({ width, height }) => {
        return (
          width > 0 && height > 0 && <SparkLineInner {...props} width={width} height={height} />
        );
      }}
    </ParentSize>
  );
}

function SparkLineInner<T extends Datum>({
  width: outerWidth,
  height: outerHeight,
  children,
  data,
  series,
  tooltipContent = (d) => series[0].valueAccessor(d).toString(),
  margin: marginProp = {
    top: 12,
    right: 0,
    bottom: 32,
    left: 0,
  },
  padding = {
    top: 0,
    bottom: 0,
  },
}: {
  width: number;
  height: number;
} & SparkLineProps<T>) {
  const [leftAxisMargin, setLeftAxisMargin] = useState<number>();

  const margin = {
    ...marginProp,
    left: marginProp.left + (leftAxisMargin ?? 0),
  };

  const width = outerWidth - margin.left - margin.right;
  const height = outerHeight - margin.top - margin.bottom;

  const { startDate, endDate } = useMemo(() => {
    const dates = data.map(({ date }) => date);
    const times = dates.map((d) => d.getTime());

    return {
      startDate: dates[times.indexOf(Math.min(...times))],
      endDate: dates[times.indexOf(Math.max(...times))],
    };
  }, [data]);

  const { minY, maxY } = useMemo(() => {
    const values = series
      .filter(({ isActive }) => isActive !== false)
      .flatMap(({ valueAccessor }) => data.map((d) => valueAccessor(d)))
      .filter((v): v is number => v != null);

    return {
      minY: Math.min(...values),
      maxY: Math.max(...values),
    };
  }, [data, series]);

  const { yScale, xScale } = useMemo(() => {
    const rangeY = maxY - minY;
    return {
      yScale: scaleLinear<number>({
        domain: [minY - rangeY * (padding.bottom ?? 0), maxY + rangeY * (padding.top ?? 0)],
        range: [height, 0],
        nice: true,
        clamp: true,
      }),
      xScale: scaleUtc<number>({
        domain: [startDate, endDate],
        range: [0, width],
      }),
    };
  }, [startDate, endDate, minY, maxY, height, width, padding.bottom, padding.top]);

  const chartContext: ChartContextType<T> = {
    width,
    height,
    data,
    series,
    startDate,
    endDate,
    xScale,
    yScale,
    minY,
    maxY,
    margin,
    padding,
    tooltipContent,
    leftAxisMargin,
    setLeftAxisMargin,
  };

  const tooltipContext = useTooltip({
    seriesId: series[0].id,
    chartContext,
  });

  const {
    tooltipData,
    TooltipWrapper,
    tooltipLeft,
    tooltipTop,
    handleTooltip,
    hideTooltip,
    containerRef,
  } = tooltipContext;

  return (
    <ChartContext.Provider value={chartContext}>
      <ChartTooltipContext.Provider value={tooltipContext}>
        <svg width={outerWidth} height={outerHeight} ref={containerRef}>
          {children}
          <Group left={margin.left} top={margin.top}>
            {/* Tooltip hover line + circle */}
            {tooltipData && (
              <>
                <Line
                  x1={xScale(tooltipData.date)}
                  x2={xScale(tooltipData.date)}
                  y1={height}
                  y2={0}
                  stroke="black"
                  strokeWidth={1}
                />

                {series.map((s) => (
                  <Circle
                    key={s.id}
                    cx={xScale(tooltipData.date)}
                    cy={yScale(s.valueAccessor(tooltipData))}
                    r={2}
                    className="text-primary"
                    fill="currentColor"
                  />
                ))}
              </>
            )}

            {/* Tooltip hover region */}
            <Bar
              x={0}
              y={0}
              width={width}
              height={height}
              onTouchStart={handleTooltip}
              onTouchMove={handleTooltip}
              onMouseMove={handleTooltip}
              onMouseLeave={hideTooltip}
              fill="transparent"
            />
          </Group>
        </svg>

        {/* Tooltips */}
        <div className="absolute inset-0 pointer-events-none">
          {tooltipData && (
            <TooltipWrapper
              key={tooltipData.date.toString()}
              left={(tooltipLeft ?? 0) + margin.left}
              top={(tooltipTop ?? 0) + margin.top}
              offsetLeft={8}
              offsetTop={12}
              className="absolute"
              unstyled={true}
            >
              <div className="px-4 py-2 text-base bg-white border border-gray-200 rounded-md shadow-md pointer-events-none">
                {tooltipContent?.(tooltipData) ?? series[0].valueAccessor(tooltipData)}
              </div>
            </TooltipWrapper>
          )}
        </div>
      </ChartTooltipContext.Provider>
    </ChartContext.Provider>
  );
}
