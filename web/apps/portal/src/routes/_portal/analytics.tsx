import { createFileRoute, redirect } from "@tanstack/react-router";
import { AlertTriangle } from "lucide-react";
import { useMemo, useState } from "react";
import { computeMetrics } from "~/components/analytics/analytics-transform";
import { formatCount } from "~/components/analytics/format";
import { HeaderStat } from "~/components/analytics/header-stat";
import { useVerificationsQuery } from "~/components/analytics/hooks/queries/use-verifications-query";
import { RangeControl } from "~/components/analytics/range-control";
import {
  availableAnalyticsPeriods,
  defaultAnalyticsPeriodDays,
} from "~/components/analytics/schema/analytics.schema";
import {
  REJECTED_COLOR,
  VALID_COLOR,
  VerificationsChart,
} from "~/components/analytics/verifications-chart";
import { Alert, AlertDescription, AlertTitle } from "~/components/ui/alert";
import { Button } from "~/components/ui/button";
import { isRetentionExceededError, isUnauthorizedError } from "~/lib/portal-api";
import { canReadAnalytics, getDefaultTabHref } from "~/lib/scopes";

const CHART_HEIGHT = 280;

type CardState = "loading" | "error" | "empty" | "populated";

export const Route = createFileRoute("/_portal/analytics")({
  beforeLoad: ({ context }) => {
    if (!canReadAnalytics(context.session.scopes)) {
      throw redirect({ to: getDefaultTabHref(context.session.scopes) ?? "/" });
    }
  },
  component: AnalyticsPage,
});

function AnalyticsPage() {
  const { session, logsRetentionDays } = Route.useRouteContext();

  const periodOptions = useMemo(
    () => availableAnalyticsPeriods(logsRetentionDays),
    [logsRetentionDays],
  );
  const [days, setDays] = useState<number>(() => defaultAnalyticsPeriodDays(logsRetentionDays));

  const { buckets, isInitialLoading, isFetching, isError, error, refetch } =
    useVerificationsQuery(days);
  const metrics = useMemo(() => computeMetrics(buckets), [buckets]);

  if (isError && isUnauthorizedError(error)) {
    return (
      <main className="mx-auto max-w-5xl px-4 pt-8 pb-12 sm:px-8">
        <SessionExpired returnUrl={session.returnUrl} />
      </main>
    );
  }

  const state = cardState(isInitialLoading, isError, metrics.totalRequests);
  const stat = (count: number) => {
    if (state === "loading") {
      return null;
    }
    return state === "populated" ? formatCount(count) : "--";
  };

  return (
    <main className="mx-auto max-w-5xl px-4 pt-8 pb-12 sm:px-8">
      <header className="mb-6 flex flex-col items-start gap-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
        <div className="flex flex-col gap-1">
          <h1 className="font-semibold text-gray-12 text-xl">Analytics</h1>
          <p className="text-gray-11 text-sm">
            Monitor verification activity and usage trends for your API keys.
          </p>
        </div>
        {periodOptions.length > 1 && (
          <RangeControl options={periodOptions} value={days} onChange={setDays} />
        )}
      </header>

      <section className="rounded-lg border border-primary/10 bg-background">
        <div className="grid divide-y divide-primary/10 border-primary/10 border-b sm:grid-cols-3 sm:divide-x sm:divide-y-0">
          <HeaderStat label="Total requests" value={stat(metrics.totalRequests)} />
          <HeaderStat label="Valid" value={stat(metrics.validRequests)} swatch={VALID_COLOR} />
          <HeaderStat label="Invalid" value={stat(metrics.errorRequests)} swatch={REJECTED_COLOR} />
        </div>

        <div
          className={`p-4 transition-opacity ${isFetching && !isInitialLoading ? "opacity-60" : ""}`}
          style={{ height: CHART_HEIGHT + 32 }}
        >
          <ChartSlot
            state={state}
            message={
              isRetentionExceededError(error)
                ? "That time range isn't available. Try a shorter range."
                : "Couldn't load your analytics"
            }
            onRetry={isRetentionExceededError(error) ? undefined : () => refetch()}
          >
            <VerificationsChart buckets={buckets} days={days} />
          </ChartSlot>
        </div>
      </section>
    </main>
  );
}

function cardState(loading: boolean, failed: boolean, total: number): CardState {
  if (loading) {
    return "loading";
  }
  if (failed) {
    return "error";
  }
  return total === 0 ? "empty" : "populated";
}

function ChartSlot({
  state,
  message,
  onRetry,
  children,
}: {
  state: CardState;
  message: string;
  onRetry?: () => void;
  children: React.ReactNode;
}) {
  switch (state) {
    case "loading":
      return (
        <div
          className="h-full w-full rounded-md bg-gray-3 motion-safe:animate-pulse"
          aria-busy="true"
        />
      );
    case "error":
      return (
        <div className="flex h-full flex-col items-center justify-center gap-3">
          <p className="text-gray-12 text-sm">{message}</p>
          {onRetry && (
            <Button variant="outline" onClick={onRetry}>
              Try again
            </Button>
          )}
        </div>
      );
    case "empty":
      return (
        <div className="flex h-full items-center justify-center text-gray-11 text-sm">
          No data for this time range
        </div>
      );
    case "populated":
      return children;
  }
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
