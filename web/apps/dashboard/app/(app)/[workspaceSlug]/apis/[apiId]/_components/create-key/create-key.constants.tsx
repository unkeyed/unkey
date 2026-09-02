import type { StepNamesFrom } from "@unkey/ui";
import {
  IconCalendarClockOutline18,
  IconChartPieOutline18,
  IconCodeOutline18,
  IconGaugeOutline18,
  IconKey2Outline18,
  IconShieldKeyOutline18,
} from "nucleo-ui-outline-18";
import { MetadataSetup } from "@/components/dashboard/metadata/metadata-setup";
import { RatelimitSetup } from "@/components/dashboard/ratelimits/ratelimit-setup";
import { UsageSetup } from "./components/credits-setup";
import { ExpirationSetup } from "./components/expiration-setup";
import { GeneralSetup } from "./components/general-setup";
import { RbacSetup } from "./components/rbac-setup";
import type { SectionState } from "./types";

export const UNNAMED_KEY = "Unnamed Key" as const;
export const SECTIONS = [
  {
    id: "general",
    label: "General Setup",
    icon: IconKey2Outline18,
    content: () => <GeneralSetup />,
  },
  {
    id: "ratelimit",
    label: "Ratelimit",
    icon: IconGaugeOutline18,
    content: () => <RatelimitSetup />,
  },
  {
    id: "credits",
    label: "Credits",
    icon: IconChartPieOutline18,
    content: () => <UsageSetup />,
  },
  {
    id: "expiration",
    label: "Expiration",
    icon: IconCalendarClockOutline18,
    content: () => <ExpirationSetup />,
  },
  {
    id: "rbac",
    label: "Permissions",
    icon: IconShieldKeyOutline18,
    content: () => <RbacSetup />,
  },
  {
    id: "metadata",
    label: "Metadata",
    icon: IconCodeOutline18,
    content: () => <MetadataSetup entityType="key" />,
  },
] as const;

export type DialogSectionName = StepNamesFrom<typeof SECTIONS>;

export const DEFAULT_STEP_STATES: Record<DialogSectionName, SectionState> = Object.fromEntries(
  SECTIONS.map((section) => [section.id, "initial"]),
) as Record<DialogSectionName, SectionState>;
