"use client";

import type { ChartMarkers } from "@/components/charts/chart-markers";
import type { RangePreset } from "@/lib/metrics/app-metrics";
import { METRIC_VIEWS, RANGE_LABELS, formatValue, primarySeries } from "@/lib/metrics/views";
import {
  Card,
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelDescription,
  SlidePanelHeader,
  SlidePanelTitle,
} from "@unkey/ui";
import { absoluteScale } from "./allocation";
import { ChartHoverProvider } from "./chart-hover";
import { commitSubject, sourceLabel } from "./deploy-identity";
import { EffectChips } from "./effect-chips";
import type { DeployView, PanelModel, PanelStats } from "./model";
import { SeriesChart } from "./series-chart";

const PANEL_SYNC_ID = "app-metrics-panel";

const STATS: Array<{ key: keyof PanelStats; label: string }> = [
  { key: "latest", label: "Latest" },
  { key: "average", label: "Average" },
  { key: "peak", label: "Peak" },
  { key: "low", label: "Low" },
];

type Props = {
  panel: PanelModel | null;
  environmentSlug: string;
  range: RangePreset;
  markers: ChartMarkers;
  onClose: () => void;
};

export function MetricPanel({ panel, environmentSlug, range, markers, onClose }: Props) {
  return (
    <SlidePanel isOpen={panel !== null} onClose={onClose} widthClassName="w-[720px]" backdrop="dim">
      {panel && (
        <ChartHoverProvider>
          <PanelBody
            panel={panel}
            subtitle={`${environmentSlug} · ${RANGE_LABELS[range].toLowerCase()}`}
            markers={markers}
          />
        </ChartHoverProvider>
      )}
    </SlidePanel>
  );
}

function PanelBody({
  panel,
  subtitle,
  markers,
}: {
  panel: PanelModel;
  subtitle: string;
  markers: ChartMarkers;
}) {
  const view = METRIC_VIEWS[panel.card.metric];
  const primary = primarySeries(panel.card.metric);
  const hasManySeries = Object.keys(view.series).length > 1;
  return (
    <>
      <SlidePanelHeader>
        <div className="flex flex-col gap-1">
          <SlidePanelTitle>{view.title}</SlidePanelTitle>
          <SlidePanelDescription className="first-letter:uppercase">
            {subtitle}
          </SlidePanelDescription>
        </div>
        <SlidePanelCloseButton />
      </SlidePanelHeader>
      <SlidePanelContent className="flex flex-col gap-6 overflow-y-auto px-6 pt-2 pb-6">
        <Card>
          <dl className="grid grid-cols-4 border-b border-grayA-4">
            {STATS.map(({ key, label }) => (
              <div
                key={key}
                className="flex flex-col gap-1 border-l border-grayA-4 px-4 py-2.5 first:border-l-0"
              >
                <dt className="text-xs font-medium text-gray-10">
                  {label}
                  {hasManySeries && (
                    <span className="font-normal text-gray-9"> · {primary.label}</span>
                  )}
                </dt>
                <dd className="font-mono text-sm tabular-nums text-gray-12">
                  {panel.stats ? formatValue(view.unit, panel.stats[key]) : "—"}
                </dd>
              </div>
            ))}
          </dl>
          <div className="px-4 pt-3 pb-3">
            <SeriesChart
              title={view.title}
              state={panel.card.state}
              series={view.series}
              scale={absoluteScale(view.unit)}
              markers={markers}
              height={300}
              syncId={PANEL_SYNC_ID}
              connectNulls={view.gaps === "bridge"}
            />
          </div>
        </Card>
        {panel.byInstance && (
          <Card className="flex flex-col gap-1.5 px-4 pt-3 pb-3">
            <h3 className="text-sm font-medium text-gray-12">By instance</h3>
            <SeriesChart
              title={`${view.title} by instance`}
              state={panel.byInstance.chart}
              series={panel.byInstance.series}
              scale={absoluteScale(view.unit)}
              height={160}
              syncId={PANEL_SYNC_ID}
            />
          </Card>
        )}
        {panel.deploys.length > 0 && (
          <section className="flex flex-col gap-2">
            <h3 className="text-sm font-medium text-gray-12">Deploys in range</h3>
            <ul className="flex flex-col">
              {panel.deploys.map((deploy) => (
                <DeployItem key={deploy.marker.deploymentId} deploy={deploy} />
              ))}
            </ul>
          </section>
        )}
      </SlidePanelContent>
    </>
  );
}

function DeployItem({ deploy }: { deploy: DeployView }) {
  return (
    <li className="flex items-center gap-3 border-t border-grayA-4 py-2 text-xs">
      <span className="shrink-0 font-mono text-gray-10">
        {sourceLabel(deploy.marker.source, deploy.marker.deploymentId)}
      </span>
      <span className="min-w-0 flex-1 truncate text-gray-12">
        {commitSubject(deploy.marker.source)}
      </span>
      <EffectChips effects={deploy.effects} className="shrink-0" />
    </li>
  );
}
