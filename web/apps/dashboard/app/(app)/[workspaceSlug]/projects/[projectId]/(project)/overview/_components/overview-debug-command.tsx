"use client";

import {
  SCENARIO_LABELS,
  type Scenario,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/mock-data";
import {
  type CmdGroup,
  PrototypeCommandPalette,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/scenario";

const SCENARIOS: Scenario[] = ["new", "migrated", "active"];

export function OverviewDebugCommand({
  scenario,
  onScenario,
  onReset,
}: {
  scenario: Scenario;
  onScenario: (s: Scenario) => void;
  onReset: () => void;
}) {
  const groups: CmdGroup[] = [
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
