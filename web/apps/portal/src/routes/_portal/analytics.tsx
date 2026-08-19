import { createFileRoute, redirect } from "@tanstack/react-router";
import { AlertTriangle, BarChart3 } from "lucide-react";
import { useMemo, useState } from "react";
import { computeMetrics } from "~/components/analytics/analytics-transform";
import { useVerificationsQuery } from "~/components/analytics/hooks/queries/use-verifications-query";
import {
  availableAnalyticsPeriods,
  defaultAnalyticsPeriodDays,
} from "~/components/analytics/schema/analytics.schema";
import { VerificationsChart } from "~/components/analytics/verifications-chart";
import { Alert, AlertDescription, AlertTitle } from "~/components/ui/alert";
import { Button } from "~/components/ui/button";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "~/components/ui/empty";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "~/components/ui/select";
import { isRetentionExceededError, isUnauthorizedError } from "~/lib/portal-api";

export const Route = createFileRoute("/_portal/analytics")({
  beforeLoad: () => {
    // Analytics is deferred to v2. The route is kept for reuse but blocked at
    // the route layer: it must not render even for a session that carries
    // analytics:read, and direct navigation is redirected away.
    throw redirect({ to: "/keys" });
  },
  component: AnalyticsPage,
});

function AnalyticsPage() {
  const { portalConfig, logsRetentionDays } = Route.useRouteContext();

  // Only offer windows the workspace can actually query; a longer one than its
  // retention would just error server-side. Retention is fixed for the session,
  // so compute the options and initial window once.
  const periodOptions = useMemo(
    () => availableAnalyticsPeriods(logsRetentionDays),
    [logsRetentionDays],
  );
  const [days, setDays] = useState<number>(() => defaultAnalyticsPeriodDays(logsRetentionDays));

  const { buckets, isInitialLoading, isError, error, refetch } = useVerificationsQuery(days);
  const metrics = useMemo(() => computeMetrics(buckets), [buckets]);

  return (
    <main className="mx-auto max-w-5xl px-4 pt-8 pb-12 sm:px-8">
      <header className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
        <div className="flex flex-col gap-1">
          <h1 className="font-semibold text-gray-12 text-xl">Analytics</h1>
          <p className="text-gray-11 text-sm">
            Monitor verification activity and usage trends for your API keys.
          </p>
        </div>
        {periodOptions.length > 1 && (
          <Select
            value={String(days)}
            onValueChange={(value) => setDays(Number(value))}
            items={periodOptions.map((option) => ({
              value: String(option.days),
              label: option.label,
            }))}
          >
            <SelectTrigger className="w-44" aria-label="Time period">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {periodOptions.map((option) => (
                <SelectItem key={option.days} value={String(option.days)}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
      </header>

      {isInitialLoading ? (
        <AnalyticsLoading />
      ) : isError && isUnauthorizedError(error) ? (
        // Expired/invalid session: retrying won't help — send the user back to
        // the application that launched the portal.
        <SessionExpired returnUrl={portalConfig?.returnUrl ?? null} />
      ) : isError && isRetentionExceededError(error) ? (
        // Window isn't available (e.g. retention lowered mid-session). Options
        // are gated to what's available, so this is rare; show a neutral,
        // actionable message rather than the generic error.
        <AnalyticsError
          title="Time range unavailable"
          message="That time range isn't available. Try a shorter range."
          onRetry={refetch}
        />
      ) : isError ? (
        <AnalyticsError
          message={error instanceof Error ? error.message : undefined}
          onRetry={refetch}
        />
      ) : metrics.totalRequests === 0 ? (
        <AnalyticsEmpty />
      ) : (
        <div className="flex flex-col gap-6">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <MetricCard label="Total Requests" value={metrics.totalRequests.toLocaleString()} />
            <MetricCard label="Success Rate" value={formatPercent(metrics.successRate)} />
            <MetricCard label="Error Rate" value={formatPercent(metrics.errorRate)} />
          </div>
          <div className="rounded-lg border border-primary/10 bg-background p-4">
            <VerificationsChart buckets={buckets} days={days} />
          </div>
        </div>
      )}
    </main>
  );
}

/** Render a [0, 1] fraction as a one-decimal percentage. */
function formatPercent(fraction: number): string {
  return `${(fraction * 100).toFixed(1)}%`;
}

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-primary/10 bg-background p-4">
      <span className="text-gray-11 text-xs">{label}</span>
      <div className="mt-1 font-semibold text-2xl text-gray-12 tabular-nums">{value}</div>
    </div>
  );
}

function AnalyticsLoading() {
  return (
    <div
      className="flex min-h-64 items-center justify-center text-gray-11 text-sm"
      aria-busy="true"
    >
      Loading analytics…
    </div>
  );
}

function AnalyticsEmpty() {
  return (
    <div className="rounded-lg border border-primary/10 bg-background">
      <Empty>
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <BarChart3 />
          </EmptyMedia>
          <EmptyTitle>No verification data yet</EmptyTitle>
          <EmptyDescription>
            Once your keys start being verified, usage metrics and trends will appear here.
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    </div>
  );
}

function SessionExpired({ returnUrl }: { returnUrl: string | null }) {
  return (
    <div className="flex flex-col items-center gap-4">
      <Alert className="max-w-md">
        <AlertTriangle />
        <AlertTitle>Your session has expired</AlertTitle>
        <AlertDescription>Return to your application to continue.</AlertDescription>
      </Alert>
      {returnUrl && (
        <Button variant="outline" render={<a href={returnUrl}>Back to application</a>} />
      )}
    </div>
  );
}

function AnalyticsError({
  title = "Couldn't load your analytics",
  message,
  onRetry,
}: {
  title?: string;
  message?: string;
  onRetry: () => void;
}) {
  return (
    <div className="flex flex-col items-center gap-4">
      <Alert variant="destructive" className="max-w-md">
        <AlertTriangle />
        <AlertTitle>{title}</AlertTitle>
        <AlertDescription>{message ?? "Something went wrong. Please try again."}</AlertDescription>
      </Alert>
      <Button variant="outline" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}
