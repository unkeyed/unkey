"use client";

import { AnimatedColumn } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/_components/flow/animated-column";
import { CardFrame } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/_components/flow/card";
import {
  FlowSplit,
  FlowViewport,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/_components/flow/flow-layout";
import {
  type StepNavModel,
  StepNavView,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/_components/flow/step-nav";
import {
  stepLabel,
  stepStatus,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/_components/flow/steps";
import {
  IconCodeBranchOutline18,
  IconDatabaseOutline18,
  IconLayers3Outline18,
  type IconProps,
  IconSquareBulletListOutline18,
  Planetscale,
} from "@unkey/icons";
import { Button } from "@unkey/ui";
import Link from "next/link";
import type { ComponentType, ReactNode } from "react";
import {
  type ConnectedFlowView,
  type SourceChoice,
  type SourceStep,
  type Step,
  showsAccount,
  sourceFlow,
  stepCopy,
} from "./create-flow-model";
import { BranchesPane, ReviewExistingPane, SelectPane } from "./existing-database-panes";
import { ConfigurePane, EnvironmentsPane, ReviewNewPane } from "./new-database-panes";
import { AccountMenu, ConnectPane, SourcePane } from "./panes";
import { type DatabaseFlow, useCreateDatabaseFlow } from "./use-create-database-flow";

const stepIcon: Record<Step, ComponentType<IconProps>> = {
  connect: Planetscale,
  source: IconLayers3Outline18,
  database: IconDatabaseOutline18,
  environments: IconCodeBranchOutline18,
  review: IconSquareBulletListOutline18,
};

export function CreateDatabaseFlow() {
  const flow = useCreateDatabaseFlow();
  const { order, previous, step, next } = flow.view;
  return (
    <FlowViewport>
      <FlowSplit steps={<StepNavView nav={stepNav(flow)} />}>
        <AnimatedColumn
          at={order}
          previous={previous}
          current={step}
          next={next}
          renderCard={(id) => <StepCard flow={flow} step={id} />}
        />
      </FlowSplit>
    </FlowViewport>
  );
}

function stepNav({ view, goTo }: DatabaseFlow): StepNavModel<Step> {
  return {
    steps: view.steps.map((id, index) => ({
      id,
      status: stepStatus(index, view.at),
      title: stepCopy[id].nav,
      detail: view.details?.[id] ?? null,
      icon: stepIcon[id],
    })),
    goTo,
  };
}

function StepCard({ flow, step }: { flow: DatabaseFlow; step: Step }) {
  const { steps } = flow.view;
  const { source } = flow.state;
  const index = steps.indexOf(step);
  const previous = steps[index - 1];
  return (
    <CardFrame
      copy={{ ...stepCopy[step], ...(source && sourceFlow[source].copy[step]) }}
      eyebrow={stepLabel(index, steps.length)}
      action={
        showsAccount[step] && flow.view.connected ? (
          <AccountMenu org={flow.view.org} onDisconnect={flow.disconnect} />
        ) : null
      }
      back={
        previous ? (
          <Button variant="outline" size="sm" onClick={() => flow.goTo(previous)}>
            Back
          </Button>
        ) : (
          <Button variant="outline" size="sm" render={<Link href={flow.cancelHref} />}>
            Cancel
          </Button>
        )
      }
    >
      <StepPane flow={flow} step={step} />
    </CardFrame>
  );
}

function StepPane({ flow, step }: { flow: DatabaseFlow; step: Step }) {
  const { view, state, dispatch } = flow;
  if (step === "connect") {
    return <ConnectPane connecting={flow.connecting} onConnect={flow.connect} />;
  }
  if (!view.connected) {
    return null;
  }
  if (step === "source") {
    return (
      <SourcePane
        source={state.source}
        org={view.org}
        orgs={view.orgs}
        onSource={(source) => dispatch({ type: "source", source })}
        onOrg={(org) => dispatch({ type: "org", org })}
        onContinue={flow.advance}
      />
    );
  }
  return state.source === null ? null : sourcePanes[state.source][step](flow, view);
}

type ConnectedPane = (flow: DatabaseFlow, view: ConnectedFlowView) => ReactNode;

const sourcePanes: Record<SourceChoice, Record<SourceStep, ConnectedPane>> = {
  new: {
    database: ({ state, dispatch, advance }, view) => (
      <ConfigurePane
        config={state.config}
        nameError={view.nameError}
        canContinue={view.nameValid}
        onChange={(config) => dispatch({ type: "config", config })}
        onContinue={advance}
      />
    ),
    environments: ({ state, dispatch, advance }, view) => (
      <EnvironmentsPane
        org={view.org}
        engine={state.config.engine}
        plan={state.plan}
        compute={state.compute}
        total={view.total}
        hasCard={view.hasCard}
        onPlan={(plan) => dispatch({ type: "plan", plan })}
        onCompute={(env, compute) => dispatch({ type: "compute", env, compute })}
        onContinue={advance}
      />
    ),
    review: ({ state, submitting, submit }, view) => (
      <ReviewNewPane
        org={view.org}
        config={state.config}
        branches={view.newBranches}
        total={view.total}
        submitting={submitting}
        onSubmit={submit}
      />
    ),
  },
  existing: {
    database: ({ dispatch, advance }, view) => (
      <SelectPane
        databases={view.linkable}
        org={view.org}
        selected={view.existing?.database ?? null}
        onSelect={(database) => dispatch({ type: "select", database })}
        onContinue={advance}
      />
    ),
    environments: ({ state, dispatch, advance }, view) =>
      view.existing ? (
        <BranchesPane
          database={view.existing.database}
          plan={state.plan}
          branches={view.existing.branches}
          onPlan={(plan) => dispatch({ type: "plan", plan })}
          onBranches={(branches) => dispatch({ type: "branches", branches })}
          onContinue={advance}
        />
      ) : null,
    review: ({ submitting, submit }, view) =>
      view.existing ? (
        <ReviewExistingPane
          database={view.existing.database}
          branches={view.existing.linkedBranches}
          submitting={submitting}
          onSubmit={submit}
        />
      ) : null,
  },
};
