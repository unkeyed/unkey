import type { PlanFeatureKind } from "@/lib/billing/plan-features";
import {
  IconClockRotateClockwiseOutline18,
  IconEarthOutline18,
  IconLayers3Outline18,
  IconMicrochipOutline18,
  type IconProps,
  IconRamOutline18,
  IconUserOutline18,
} from "@unkey/icons";
import type { ComponentType } from "react";

export const FEATURE_ICONS: Record<PlanFeatureKind, ComponentType<IconProps>> = {
  team: IconUserOutline18,
  cpu: IconMicrochipOutline18,
  memory: IconRamOutline18,
  domains: IconEarthOutline18,
  autoscale: IconLayers3Outline18,
  logs: IconClockRotateClockwiseOutline18,
};
