"use client";

import {
  SCENARIO_LABELS,
  type Scenario,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/mock-data";
import {
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
  onReset,
}: {
  scenario: Scenario;
  onScenario: (s: Scenario) => void;
  layout: OverviewLayout;
  onLayout: (l: OverviewLayout) => void;
  onReset: () => void;
}) {
  const groups: CmdGroup[] = [
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
