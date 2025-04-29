import type { StepNamesFrom } from "@/components/dialog-container/navigable-dialog";
import { CalendarClock, ChartPie, Code, Gauge, Key2 } from "@unkey/icons";
import type { SectionState } from "./types";

import { UsageSetup } from "./components/credits-setup";
import { ExpirationSetup } from "./components/expiration-setup";
import { GeneralSetup } from "./components/general-setup";
import { MetadataSetup } from "./components/metadata-setup";
import { RatelimitSetup } from "./components/ratelimit-setup";

export const UNNAMED_KEY = "Unnamed Key" as const;
export const SECTIONS = [
  {
    id: "general",
    label: "General Setup",
    icon: Key2,
    content: () => <GeneralSetup />,
  },
  {
    id: "ratelimit",
    label: "Ratelimit",
    icon: Gauge,
    content: () => <RatelimitSetup />,
  },
  {
    id: "credits",
    label: "Credits",
    icon: ChartPie,
    content: () => <UsageSetup />,
  },
  {
    id: "expiration",
    label: "Expiration",
    icon: CalendarClock,
    content: () => <ExpirationSetup />,
  },
  {
    id: "metadata",
    label: "Metadata",
    icon: Code,
    content: () => <MetadataSetup />,
  },
] as const;

export type DialogSectionName = StepNamesFrom<typeof SECTIONS>;

export const DEFAULT_STEP_STATES: Record<DialogSectionName, SectionState> = Object.fromEntries(
  SECTIONS.map((section) => [section.id, "initial"]),
) as Record<DialogSectionName, SectionState>;
