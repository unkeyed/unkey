"use client";

import {
  type FlowStep,
  type StepNavModel,
  StepNavView,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/_components/flow/step-nav";
import { useProjectApps } from "@/lib/queries/apps";
import {
  Github,
  IconCubeOutline18,
  type IconProps,
  IconShippingContainerOutline18,
  IconSlidersOutline18,
  IconSplitOutline18,
} from "@unkey/icons";
import type { ComponentType } from "react";
import { useNewAppFlow } from "./flow";
import {
  type SourceKind,
  type Step,
  type WizardState,
  sourceCopy,
  stepTitle,
  visibleSteps,
} from "./wizard-model";

type Icon = ComponentType<IconProps>;

const sourceIcon: Record<SourceKind, Icon> = { git: Github, oci: IconShippingContainerOutline18 };

function stepIcons(source: SourceKind | null): Record<Step, Icon> {
  return {
    method: IconSplitOutline18,
    source: source ? sourceIcon[source] : IconCubeOutline18,
    settings: IconSlidersOutline18,
  };
}

function useStepViews(projectId: string, state: WizardState): FlowStep<Step>[] {
  const appId = state.appId ?? "";
  const app = useProjectApps(projectId).data?.find((a) => a.id === appId);
  const { source } = state;
  const chosen: Record<Step, string | null> = {
    method: source ? sourceCopy[source].sourceLabel : null,
    source: app ? (app.repositoryFullName ?? app.imageReference) : null,
    settings: null,
  };
  const icons = stepIcons(source);
  return visibleSteps(state).map((row) => ({
    ...row,
    icon: icons[row.id],
    title: stepTitle[row.id],
    detail: chosen[row.id],
  }));
}

function useStepNav(): StepNavModel<Step> {
  const { projectId, state, locked, dispatch } = useNewAppFlow();
  const steps = useStepViews(projectId, state);
  const goTo = (step: Step) => {
    if (!locked) {
      dispatch({ type: "go", step });
    }
  };
  return { steps, goTo };
}

export function StepNav() {
  return <StepNavView nav={useStepNav()} />;
}
