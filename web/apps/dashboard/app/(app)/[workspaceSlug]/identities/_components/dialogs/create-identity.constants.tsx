import { MetadataSetup } from "@/components/dashboard/metadata/metadata-setup";
import { RatelimitSetup } from "@/components/dashboard/ratelimits/ratelimit-setup";
import type { StepNamesFrom } from "@unkey/ui";
import {
  IconCodeOutline18,
  IconFingerprintOutline18,
  IconGaugeOutline18,
} from "nucleo-ui-outline-18";
import { GeneralSetup } from "./general-setup";

export const SECTIONS = [
  {
    id: "general",
    label: "General Setup",
    icon: IconFingerprintOutline18,
    content: () => <GeneralSetup />,
  },
  {
    id: "ratelimit",
    label: "Ratelimit",
    icon: IconGaugeOutline18,
    content: () => <RatelimitSetup entityType="identity" />,
  },
  {
    id: "metadata",
    label: "Metadata",
    icon: IconCodeOutline18,
    content: () => <MetadataSetup entityType="identity" />,
  },
] as const;

export type DialogSectionName = StepNamesFrom<typeof SECTIONS>;
