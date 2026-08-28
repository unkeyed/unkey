import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { DeliveryOverview, type LogdrainSeries } from "./charts";

vi.mock("@unkey/icons", () => ({
  ChartActivity: () => null,
  TimeClock: () => null,
  TriangleWarning2: () => null,
}));

vi.mock("@/components/charts/area-timeseries", () => ({
  AreaTimeseriesChart: ({ axis }: { axis: { x?: { domain?: [number, number] } } | null }) => (
    <div data-testid="chart" data-domain={JSON.stringify(axis?.x?.domain)} />
  ),
}));

describe("DeliveryOverview", () => {
  it("keeps zero-value buckets in the chart time domain", () => {
    const series = [metric(1_000, 0), metric(2_000, 10), metric(3_000, 0)] satisfies LogdrainSeries;

    const { getAllByTestId } = render(
      <DeliveryOverview series={series} loading={false} error={false} />,
    );

    for (const chart of getAllByTestId("chart")) {
      expect(chart.getAttribute("data-domain")).toBe("[1000,3000]");
    }
  });
});

function metric(ts: number, eventsDelivered: number): LogdrainSeries[number] {
  return {
    ts,
    successCount: eventsDelivered > 0 ? 1 : 0,
    transientErrorCount: 0,
    permanentErrorCount: 0,
    eventsDelivered,
    avgDurationMs: eventsDelivered > 0 ? 86 : 0,
  };
}
