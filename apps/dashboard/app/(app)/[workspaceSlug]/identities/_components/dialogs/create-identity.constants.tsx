import { MetadataSetup } from "@/components/dashboard/metadata/metadata-setup";
import { RatelimitSetup } from "@/components/dashboard/ratelimits/ratelimit-setup";
import { Code, Fingerprint, Gauge } from "@unkey/icons";
import type { StepNamesFrom } from "@unkey/ui";
import { GeneralSetup } from "./general-setup";

export const SECTIONS = [
  {
    id: "general",
    label: "General Setup",
    icon: Fingerprint,
    content: () => <GeneralSetup />,
  },
  {
    id: "ratelimit",
    label: "Ratelimit",
    icon: Gauge,
    content: () => <RatelimitSetup entityType="identity" />,
  },
  {
    id: "metadata",
    label: "Metadata",
    icon: Code,
    content: () => <MetadataSetup entityType="identity" />,
  },
] as const;

export type DialogSectionName = StepNamesFrom<typeof SECTIONS>;
