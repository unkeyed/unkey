import { ChartSlot } from "~/components/analytics/chart-slot";
import { MetricsStrip } from "~/components/analytics/metrics-strip";
import { OutcomesChart } from "~/components/analytics/outcomes-chart";
import { TimeFilter } from "~/components/analytics/time-filter";
import { FilterBar } from "~/components/keys-table/filters/filter-bar";
import { KeyActions } from "~/components/keys-table/key-actions";
import { KeysUsageTable } from "~/components/keys-table/keys-usage-table";
import { RotateKeyDialog } from "~/components/keys-table/rotate-key-dialog";
import { TooltipProvider } from "~/components/ui/tooltip";
import { useKeysPage } from "~/hooks/use-keys-page";
import { KeysError } from "./keys-error";
import { SessionExpired } from "./session-expired";

export function KeysPage() {
  const view = useKeysPage();

  if (view.status === "session-expired") {
    return (
      <main className="mx-auto w-full max-w-6xl px-4 pt-8 pb-12 sm:px-8">
        <SessionExpired returnUrl={view.returnUrl} />
      </main>
    );
  }

  const { title, filters, time, analytics, table, rotate, canRotate } = view;

  return (
    <TooltipProvider delay={300}>
      <main className="mx-auto w-full max-w-6xl px-4 pt-8 pb-12 sm:px-8">
        <header className="mb-3">
          <h1 className="font-semibold text-gray-12 text-xl">{title}</h1>
        </header>

        <div className="sticky top-0 z-20 bg-background py-1">
          <div className="flex flex-wrap items-center gap-2">
            <FilterBar {...filters} />
            {time && (
              <div className="ml-auto">
                <TimeFilter
                  presets={time.presets}
                  value={time.preset}
                  isDefault={time.preset.id === time.defaultPresetId}
                  onChange={time.onPreset}
                />
              </div>
            )}
          </div>
        </div>

        <section className="mt-3 rounded-lg border border-primary/10">
          {analytics && (
            <>
              <MetricsStrip metrics={analytics.metrics} state={analytics.chart} />
              <div
                // The chart is short enough that its tooltip overhangs the
                // table, so the stacking context has to sit above it.
                className={`relative z-10 h-44 px-4 pt-4 pb-3 transition-opacity ${
                  analytics.dimmed ? "opacity-60" : ""
                }`}
              >
                <ChartSlot state={analytics.chart}>
                  <OutcomesChart buckets={analytics.series} days={analytics.days} />
                </ChartSlot>
              </div>
            </>
          )}

          <div className={`pt-4 transition-opacity ${table.dimmed ? "opacity-60" : ""}`}>
            {table.error ? (
              <KeysError message={table.error.message} onRetry={table.error.onRetry} />
            ) : (
              <KeysUsageTable
                key={table.resetKey}
                rows={table.rows}
                isLoading={table.isLoading}
                showUsage={table.showUsage}
                onClearFilters={table.onClearFilters}
                renderActions={
                  canRotate
                    ? (key) => <KeyActions apiKey={key} onRotate={() => rotate.start(key)} />
                    : undefined
                }
              />
            )}
          </div>
        </section>

        <RotateKeyDialog rotate={rotate} />
      </main>
    </TooltipProvider>
  );
}
