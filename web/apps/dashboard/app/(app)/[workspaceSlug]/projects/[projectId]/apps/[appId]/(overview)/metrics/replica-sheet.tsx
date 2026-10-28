"use client";

import type { ChartMarkers } from "@/components/charts/chart-markers";
import { RegionFlag } from "@/components/region-flag";
import { summarizeExit } from "@/lib/collections/deploy/exit-reasons";
import { instanceLabel } from "@/lib/collections/deploy/instance-status";
import { plural } from "@/lib/fmt";
import { METRIC_VIEWS } from "@/lib/metrics/views";
import { regionInfo } from "@/lib/regions";
import type { LastExit } from "@/lib/types/deploy";
import { match } from "@unkey/match";
import {
  Card,
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelHeader,
  SlidePanelTitle,
  TimestampInfo,
} from "@unkey/ui";
import type { ReactNode } from "react";
import { LastExitText } from "../../components/last-exit";
import { absoluteScale } from "./allocation";
import { ChartHoverProvider } from "./chart-hover";
import { DeployIdentity } from "./deploy-identity";
import {
  REPLICA_METRICS,
  type ReplicaSection,
  type ReplicaSheetModel,
  type ReplicaUsage,
} from "./replica-model";
import { SeriesChart } from "./series-chart";

const SYNC_ID = "replica-sheet";
const CHART_HEIGHT = 120;

type Props = {
  sheet: ReplicaSheetModel | null;
  markers: ChartMarkers;
  onClose: () => void;
};

export function ReplicaSheet({ sheet, markers, onClose }: Props) {
  return (
    <SlidePanel
      isOpen={sheet !== null}
      onClose={onClose}
      widthClassName="w-[600px]"
      backdrop="none"
    >
      {sheet && (
        <>
          <SlidePanelHeader>
            <SlidePanelTitle className="truncate">{sheet.name}</SlidePanelTitle>
            <SlidePanelCloseButton />
          </SlidePanelHeader>
          <SlidePanelContent className="flex flex-col gap-6 overflow-y-auto px-6 pt-2 pb-6">
            <dl className="flex flex-col divide-y divide-grayA-4 rounded-lg border border-grayA-4">
              <DetailRow label="Status">
                <span className="text-gray-12">{instanceLabel[sheet.status]}</span>
              </DetailRow>
              <DetailRow label="Region">
                <span className="flex items-center gap-1.5 text-gray-12">
                  <RegionFlag region={sheet.region} size="xs" shape="circle" />
                  {regionInfo(sheet.region).city}
                  <span className="font-mono text-xs text-gray-10">{sheet.region}</span>
                </span>
              </DetailRow>
              <DetailRow label="Deployment">
                <DeployIdentity
                  source={sheet.deployment.source}
                  deploymentId={sheet.deployment.id}
                  className="min-w-0 gap-1"
                />
              </DetailRow>
              {sheet.lastExit && <LastExitRow lastExit={sheet.lastExit} />}
            </dl>
            {sheet.stopped && sheet.usage.status === "ready" && (
              <p className="text-xs text-gray-11">
                This instance has stopped. Its last recorded usage is below.
              </p>
            )}
            <ChartHoverProvider>
              <ReplicaUsageBody usage={sheet.usage} markers={markers} />
            </ChartHoverProvider>
          </SlidePanelContent>
        </>
      )}
    </SlidePanel>
  );
}

function Section({
  heading,
  detail,
  children,
}: {
  heading: string;
  detail?: string | null;
  children: ReactNode;
}) {
  return (
    <Card className="flex flex-col gap-1.5 px-4 pt-3 pb-3">
      <div className="flex items-center justify-between gap-3">
        <h3 className="text-sm font-medium text-gray-12">
          {heading}
          {detail && <span className="font-normal text-gray-10"> · {detail}</span>}
        </h3>
      </div>
      {children}
    </Card>
  );
}

function DetailRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="grid grid-cols-[112px_minmax(0,1fr)] items-start gap-4 px-4 py-3 text-sm">
      <dt className="font-medium text-gray-10">{label}</dt>
      <dd className="min-w-0">{children}</dd>
    </div>
  );
}

function LastExitRow({ lastExit }: { lastExit: LastExit }) {
  return (
    <DetailRow label="Last exit">
      <div className="flex flex-col items-start gap-1.5">
        <LastExitText summary={summarizeExit(lastExit)} />
        <span className="text-xs tabular-nums text-gray-11">
          {plural(lastExit.restartCount, "restart")}
          {lastExit.finishedAt && (
            <>
              {" · "}
              <TimestampInfo value={lastExit.finishedAt} displayType="relative" />
            </>
          )}
        </span>
      </div>
    </DetailRow>
  );
}

function ReplicaUsageBody({ usage, markers }: { usage: ReplicaUsage; markers: ChartMarkers }) {
  return match(usage)
    .with({ status: "loading" }, () =>
      REPLICA_METRICS.map((metric) => (
        <UsageSection
          key={metric}
          section={{
            metric,
            chart: { status: "loading" },
            scale: absoluteScale(METRIC_VIEWS[metric].unit),
            allocation: null,
          }}
          markers={markers}
        />
      )),
    )
    .with({ status: "failed" }, () => (
      <p className="text-xs text-gray-10">Usage is unavailable right now.</p>
    ))
    .with({ status: "none" }, () => (
      <EmptyState className="flex-1">
        <EmptyStateHeader>
          <EmptyStateTitle>No data in this range</EmptyStateTitle>
          <EmptyStateDescription>
            This instance sent no CPU, memory, disk or network data in the selected range.
          </EmptyStateDescription>
        </EmptyStateHeader>
      </EmptyState>
    ))
    .with({ status: "ready" }, ({ sections }) =>
      sections.map((section) => (
        <UsageSection key={section.metric} section={section} markers={markers} />
      )),
    )
    .exhaustive();
}

function UsageSection({ section, markers }: { section: ReplicaSection; markers: ChartMarkers }) {
  const view = METRIC_VIEWS[section.metric];
  return (
    <Section heading={view.title} detail={section.allocation}>
      <SeriesChart
        title={view.title}
        state={section.chart}
        series={view.series}
        scale={section.scale}
        markers={markers}
        height={CHART_HEIGHT}
        syncId={SYNC_ID}
      />
    </Section>
  );
}
