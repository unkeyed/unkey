import { render } from "@testing-library/react";
import type { ComponentProps, PropsWithChildren } from "react";
import { describe, expect, it, vi } from "vitest";
import { AreaTimeseriesChart } from "./area-timeseries";

vi.mock("recharts", () => ({
  Area: ({
    dataKey,
    fill,
    stroke,
    strokeDasharray,
  }: {
    dataKey: string;
    fill: string;
    stroke: string;
    strokeDasharray?: string;
  }) => (
    <path
      data-testid="area"
      data-data-key={dataKey}
      data-fill={fill}
      data-stroke={stroke}
      data-stroke-dasharray={strokeDasharray}
    />
  ),
  AreaChart: ({ children, data }: PropsWithChildren<{ data: unknown }>) => (
    <svg data-testid="area-chart" data-chart-data={JSON.stringify(data)}>
      {children}
    </svg>
  ),
  CartesianGrid: () => null,
  ReferenceLine: ({
    x,
    label,
    strokeDasharray,
  }: {
    x: number;
    label?: { value: string; content: unknown };
    strokeDasharray: string;
  }) => (
    <line
      data-testid="reference-line"
      data-x={x}
      data-label={label?.value}
      data-has-label-content={typeof label?.content === "function"}
      strokeDasharray={strokeDasharray}
    />
  ),
  XAxis: ({ domain, hide }: { domain: unknown; hide: boolean }) => (
    <g data-testid="x-axis" data-domain={JSON.stringify(domain)} data-hidden={hide} />
  ),
  YAxis: ({ domain, hide, width }: { domain: unknown; hide: boolean; width: number }) => (
    <g
      data-testid="y-axis"
      data-domain={JSON.stringify(domain)}
      data-hidden={hide}
      data-width={width}
    />
  ),
}));

vi.mock("@/components/ui/chart", () => ({
  ChartContainer: ({ children }: PropsWithChildren) => <>{children}</>,
  ChartTooltip: () => null,
}));

describe("AreaTimeseriesChart axis configuration", () => {
  it("renders a flat line for zero values when requested", () => {
    const { container, queryByText } = render(
      <AreaTimeseriesChart
        data={[
          { originalTimestamp: 1_000, errors: 0 },
          { originalTimestamp: 2_000, errors: 0 },
        ]}
        config={{ errors: { label: "Errors", color: "red" } }}
        axis={null}
        showZeroLine
      />,
    );

    expect(queryByText("No activity yet")).toBeNull();
    expect(container.querySelector('[data-testid="y-axis"]')?.getAttribute("data-domain")).toBe(
      "[0,1]",
    );
  });

  it("adds ten percent headroom to the observed maximum by default", () => {
    const { container } = render(
      <AreaTimeseriesChart
        data={[
          { originalTimestamp: 1_000, requests: 5 },
          { originalTimestamp: 2_000, requests: 10 },
        ]}
        config={{ requests: { label: "Requests", color: "blue" } }}
        axis={{ y: { floor: 0, width: 72 } }}
      />,
    );

    const yAxis = container.querySelector('[data-testid="y-axis"]');
    expect(yAxis?.getAttribute("data-domain")).toBe("[0,11]");
    expect(yAxis?.getAttribute("data-width")).toBe("72");
  });

  it("retains configured domains when the axes are hidden", () => {
    const domain: [number, number] = [1_000, 2_000];
    const props = {
      data: [
        { originalTimestamp: 1_000, requests: 0.02 },
        { originalTimestamp: 2_000, requests: 0.06 },
      ],
      config: { requests: { label: "Requests", color: "blue" } },
      axis: {
        visible: false,
        x: { domain },
        y: { floor: 0 },
      },
    } satisfies ComponentProps<typeof AreaTimeseriesChart>;

    const { container } = render(<AreaTimeseriesChart {...props} />);
    const xAxis = container.querySelector('[data-testid="x-axis"]');
    const yAxis = container.querySelector('[data-testid="y-axis"]');

    expect(xAxis?.getAttribute("data-domain")).toBe("[1000,2000]");
    expect(xAxis?.getAttribute("data-hidden")).toBe("true");
    expect(yAxis?.getAttribute("data-domain")).toBe("[0,0.066]");
    expect(yAxis?.getAttribute("data-hidden")).toBe("true");
  });

  it("draws the segment into an incomplete bucket as a dotted line", () => {
    const { container } = render(
      <AreaTimeseriesChart
        data={[
          { originalTimestamp: 1_000, requests: 5 },
          { originalTimestamp: 2_000, requests: 10 },
          { originalTimestamp: 3_000, requests: 2 },
        ]}
        config={{ requests: { label: "Requests", color: "blue" } }}
        axis={{ y: { floor: 0 } }}
        incompleteFrom={3_000}
      />,
    );
    const chartData = JSON.parse(
      container.querySelector('[data-testid="area-chart"]')?.getAttribute("data-chart-data") ??
        "[]",
    ) as Array<Record<string, number>>;
    const dottedArea = container.querySelector('[data-stroke-dasharray="4 4"]');

    expect(dottedArea?.getAttribute("data-data-key")).toBe("__incomplete_requests");
    expect(dottedArea?.getAttribute("data-fill")).toContain("url(#");
    expect(container.querySelector('[data-data-key="requests"]')?.getAttribute("data-fill")).toBe(
      "none",
    );
    expect(chartData[1]).toMatchObject({
      requests: 10,
      __complete_requests: 10,
      __incomplete_requests: 10,
    });
    expect(chartData[2]).toMatchObject({ requests: 2, __incomplete_requests: 2 });
    expect(chartData[2]?.__complete_requests).toBeUndefined();
  });

  it("renders bucketed annotations", () => {
    const { getAllByTestId } = render(
      <AreaTimeseriesChart
        data={[
          { originalTimestamp: 1_000, requests: 5 },
          { originalTimestamp: 2_000, requests: 10 },
        ]}
        config={{ requests: { label: "Requests", color: "blue" } }}
        axis={{ y: { floor: 0 } }}
        annotations={[{ timestamp: 1_000, label: "d_example" }, { timestamp: 2_000 }]}
      />,
    );

    const lines = getAllByTestId("reference-line");
    expect(lines).toHaveLength(2);
    expect(lines.map((line) => line.getAttribute("data-x"))).toEqual(["1000", "2000"]);
    expect(lines[0]?.getAttribute("stroke-dasharray")).toBe("4 4");
    expect(lines[0]?.getAttribute("data-label")).toBe("d_example");
    expect(lines[0]?.getAttribute("data-has-label-content")).toBe("true");
    expect(lines[1]?.getAttribute("data-label")).toBeNull();
  });
});
