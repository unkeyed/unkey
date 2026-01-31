"use client";

import { PageLoading } from "@/components/dashboard/page-loading";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { CloudUp } from "@unkey/icons";
import {
  Button,
  Empty,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@unkey/ui";
import { useMemo, useState } from "react";
import { BillingNavbar } from "../billing-navbar";

type Granularity = "day" | "week" | "month";
type TimeRange = "7d" | "30d" | "90d" | "1y";

const TIME_RANGES: Record<TimeRange, { label: string; days: number }> = {
  "7d": { label: "Last 7 days", days: 7 },
  "30d": { label: "Last 30 days", days: 30 },
  "90d": { label: "Last 90 days", days: 90 },
  "1y": { label: "Last year", days: 365 },
};

interface RevenueDataPoint {
  date: string;
  revenue: number;
  invoiceCount: number;
}

export default function AnalyticsPage() {
  const workspace = useWorkspaceNavigation();
  const [timeRange, setTimeRange] = useState<TimeRange>("30d");
  const [granularity, setGranularity] = useState<Granularity>("day");

  // Use useMemo to ensure stable dates for query keys
  const { startDate, endDate } = useMemo(() => {
    const now = Date.now();
    const startDate = now - TIME_RANGES[timeRange].days * 24 * 60 * 60 * 1000;
    return { startDate, endDate: now };
  }, [timeRange]);

  const { data: revenueData, isLoading: isLoadingRevenue } =
    trpc.customerBilling.analytics.revenue.useQuery({
      startDate,
      endDate,
      granularity,
    });

  const { data: usageData, isLoading: isLoadingUsage } =
    trpc.customerBilling.analytics.usage.useQuery({
      startDate,
      endDate,
    });

  const { data: summary, isLoading: isLoadingSummary } =
    trpc.customerBilling.invoices.getSummary.useQuery();

  const exportMutation = trpc.customerBilling.analytics.export.useQuery(
    {
      startDate,
      endDate,
    },
    {
      enabled: false,
    },
  );

  const handleExport = async () => {
    try {
      const result = await exportMutation.refetch();
      if (result.data) {
        const blob = new Blob([result.data.csv], { type: "text/csv" });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = result.data.filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        toast.success("Export downloaded");
      }
    } catch {
      toast.error("Failed to export data");
    }
  };

  // Check if billing beta is enabled
  if (!workspace.betaFeatures.billing) {
    return (
      <div>
        <BillingNavbar activePage={{ href: "analytics", text: "Analytics" }} />
        <div className="p-4">
          <Empty>
            <Empty.Icon />
            <Empty.Title>Customer Billing Not Enabled</Empty.Title>
            <Empty.Description>
              Customer billing is currently in beta. Contact support to enable this feature.
            </Empty.Description>
          </Empty>
        </div>
      </div>
    );
  }

  const isLoading = isLoadingRevenue || isLoadingUsage || isLoadingSummary;

  if (isLoading) {
    return <PageLoading message="Loading analytics..." />;
  }

  // Handle empty state when there are no invoices
  if (!summary || summary.totalInvoices === 0) {
    return (
      <div>
        <BillingNavbar activePage={{ href: "analytics", text: "Analytics" }} />
        <div className="p-4">
          <Empty>
            <Empty.Icon />
            <Empty.Title>No Invoice Data</Empty.Title>
            <Empty.Description>
              There are no invoices yet. Revenue and usage data will appear once invoices are created.
            </Empty.Description>
          </Empty>
        </div>
      </div>
    );
  }

  // Calculate max revenue for chart scaling
  const maxRevenue = revenueData && revenueData.length > 0
    ? Math.max(...revenueData.map((d: RevenueDataPoint) => d.revenue), 1)
    : 1;

  return (
    <div>
      <BillingNavbar activePage={{ href: "analytics", text: "Analytics" }} />
      <div className="py-3 w-full flex items-center justify-center">
        <div className="w-[900px] flex flex-col justify-center items-center gap-5 mx-6 mt-4">
          {/* Summary Cards */}
          <div className="w-full grid grid-cols-4 gap-4">
            <div className="p-4 border rounded-lg">
              <div className="text-sm text-gray-11">Total Revenue</div>
              <div className="text-2xl font-semibold">
                ${summary ? (summary.totalRevenue / 100).toFixed(2) : "0.00"}
              </div>
            </div>
            <div className="p-4 border rounded-lg">
              <div className="text-sm text-gray-11">Pending Revenue</div>
              <div className="text-2xl font-semibold">
                ${summary ? (summary.pendingRevenue / 100).toFixed(2) : "0.00"}
              </div>
            </div>
            <div className="p-4 border rounded-lg">
              <div className="text-sm text-gray-11">Total Verifications</div>
              <div className="text-2xl font-semibold">
                {usageData?.totalVerifications.toLocaleString() ?? 0}
              </div>
            </div>
            <div className="p-4 border rounded-lg">
              <div className="text-sm text-gray-11">Total Rate Limits</div>
              <div className="text-2xl font-semibold">
                {usageData?.totalRatelimits.toLocaleString() ?? 0}
              </div>
            </div>
          </div>

          {/* Filters */}
          <div className="w-full flex justify-between items-center">
            <h2 className="text-lg font-medium">Revenue Over Time</h2>
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <span className="text-sm text-gray-11">Period:</span>
                <Select
                  value={timeRange}
                  onValueChange={(value) => setTimeRange(value as TimeRange)}
                >
                  <SelectTrigger className="w-[150px]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {Object.entries(TIME_RANGES).map(([key, { label }]) => (
                      <SelectItem key={key} value={key}>
                        {label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-sm text-gray-11">Group by:</span>
                <Select
                  value={granularity}
                  onValueChange={(value) => setGranularity(value as Granularity)}
                >
                  <SelectTrigger className="w-[120px]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="day">Day</SelectItem>
                    <SelectItem value="week">Week</SelectItem>
                    <SelectItem value="month">Month</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <Button variant="outline" onClick={handleExport}>
                <CloudUp className="w-4 h-4 mr-2" />
                Export CSV
              </Button>
            </div>
          </div>

          {/* Revenue Chart */}
          {revenueData && revenueData.length > 0 ? (
            <div className="w-full border rounded-lg p-4">
              <div className="h-64 flex items-end gap-1">
                {revenueData.map((point: RevenueDataPoint) => {
                  const height = (point.revenue / maxRevenue) * 100;
                  return (
                    <div
                      key={point.date}
                      className="flex-1 flex flex-col items-center group relative"
                    >
                      <div
                        className="w-full bg-accent-9 rounded-t hover:bg-accent-10 transition-colors"
                        style={{ height: `${Math.max(height, 2)}%` }}
                      />
                      {/* Tooltip */}
                      <div className="absolute bottom-full mb-2 hidden group-hover:block bg-gray-12 text-gray-1 text-xs px-2 py-1 rounded whitespace-nowrap z-10">
                        <div>{point.date}</div>
                        <div>${(point.revenue / 100).toFixed(2)}</div>
                        <div>{point.invoiceCount} invoices</div>
                      </div>
                    </div>
                  );
                })}
              </div>
              {/* X-axis labels */}
              <div className="flex justify-between mt-2 text-xs text-gray-11">
                <span>{revenueData[0]?.date}</span>
                <span>{revenueData[revenueData.length - 1]?.date}</span>
              </div>
            </div>
          ) : (
            <div className="w-full border rounded-lg p-8 text-center text-gray-11">
              No revenue data for the selected period
            </div>
          )}

          {/* Usage Chart */}
          <div className="w-full">
            <h2 className="text-lg font-medium mb-4">Usage Summary</h2>
            <div className="grid grid-cols-2 gap-4">
              <div className="border rounded-lg p-4">
                <div className="text-sm text-gray-11 mb-2">Verifications</div>
                <div className="h-32 flex items-center justify-center">
                  <div className="text-center">
                    <div className="text-4xl font-bold text-accent-11">
                      {usageData?.totalVerifications.toLocaleString() ?? 0}
                    </div>
                    <div className="text-sm text-gray-11 mt-1">
                      in {TIME_RANGES[timeRange].label.toLowerCase()}
                    </div>
                  </div>
                </div>
              </div>
              <div className="border rounded-lg p-4">
                <div className="text-sm text-gray-11 mb-2">Rate Limits</div>
                <div className="h-32 flex items-center justify-center">
                  <div className="text-center">
                    <div className="text-4xl font-bold text-accent-11">
                      {usageData?.totalRatelimits.toLocaleString() ?? 0}
                    </div>
                    <div className="text-sm text-gray-11 mt-1">
                      in {TIME_RANGES[timeRange].label.toLowerCase()}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Invoice Stats */}
          {summary && (
            <div className="w-full">
              <h2 className="text-lg font-medium mb-4">Invoice Status Distribution</h2>
              <div className="border rounded-lg p-4">
                <div className="flex gap-4">
                  {Object.entries(summary.statusCounts).map(([status, count]) => (
                    <div key={status} className="flex-1 text-center">
                      <div className="text-2xl font-semibold">{count as number}</div>
                      <div className="text-sm text-gray-11 capitalize">{status}</div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
