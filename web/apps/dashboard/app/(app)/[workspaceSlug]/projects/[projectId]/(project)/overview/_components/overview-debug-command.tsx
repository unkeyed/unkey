"use client";

import {
  SCENARIO_LABELS,
  type Scenario,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/mock-data";
import {
  CHART_SCHEMES,
  CHART_SCHEME_LABELS,
  type ChartScheme,
  type CmdGroup,
  OVERVIEW_LAYOUTS,
  OVERVIEW_LAYOUT_LABELS,
  type OverviewLayout,
  PrototypeCommandPalette,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/scenario";

const SCENARIOS: Scenario[] = ["new", "migrated", "active"];

export function OverviewDebugCommand({
  scenario,
  onScenario,
  layout,
  onLayout,
  chartScheme,
  onChartScheme,
  onReset,
}: {
  scenario: Scenario;
  onScenario: (s: Scenario) => void;
  layout: OverviewLayout;
  onLayout: (l: OverviewLayout) => void;
  chartScheme: ChartScheme;
  onChartScheme: (c: ChartScheme) => void;
  onReset: () => void;
}) {
  const groups: CmdGroup[] = [
    {
      name: "Chart colors",
      items: CHART_SCHEMES.map((c) => ({
        id: `chart-${c}`,
        group: "Chart colors",
        label: CHART_SCHEME_LABELS[c],
        active: c === chartScheme,
        run: () => onChartScheme(c),
      })),
    },
    {
      name: "Layout",
      items: OVERVIEW_LAYOUTS.map((l) => ({
        id: `layout-${l}`,
        group: "Layout",
        label: OVERVIEW_LAYOUT_LABELS[l],
        active: l === layout,
        run: () => onLayout(l),
      })),
    },
    {
      name: "Scenario",
      items: SCENARIOS.map((s) => ({
        id: `sc-${s}`,
        group: "Scenario",
        label: SCENARIO_LABELS[s],
        active: s === scenario,
        run: () => onScenario(s),
      })),
    },
    {
      name: "Data",
      items: [
        { id: "reset", group: "Data", label: "Reset prototype data", active: false, run: onReset },
      ],
    },
  ];

  return <PrototypeCommandPalette groups={groups} />;
}
