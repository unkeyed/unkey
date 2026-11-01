"use client";

import { useAppsInProject } from "@/lib/deploy-data";
import {
  Github,
  IconCheckOutline18,
  IconCubeOutline18,
  IconPenWriting3Outline18,
  type IconProps,
  IconShippingContainerOutline18,
  IconSlidersOutline18,
  IconSplitOutline18,
} from "@unkey/icons";
import { Button, Drawer, Tooltip, TooltipContent, TooltipTrigger, cn } from "@unkey/ui";
import { AnimatePresence, LayoutGroup, m, useReducedMotion } from "framer-motion";
import { type ComponentType, type ReactNode, useId, useState } from "react";
import { useNewAppFlow } from "./flow";
import { StepAside } from "./flow-layout";
import {
  type SourceKind,
  type Step,
  type StepRow,
  type WizardState,
  sourceCopy,
  stepProgress,
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

type StepView = StepRow & { title: string; detail: string | null; icon: Icon };

function useStepViews(projectId: string, state: WizardState): StepView[] {
  const app = useAppsInProject(projectId).data.find(({ id }) => id === state.appId);
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
    detail: row.status === "done" ? chosen[row.id] : null,
  }));
}

function useStepNav() {
  const { projectId, state, locked, dispatch } = useNewAppFlow();
  const steps = useStepViews(projectId, state);
  const goTo = (step: StepView) => {
    if (step.status === "done" && !locked) {
      dispatch({ type: "go", step: step.id });
    }
  };
  return { steps, goTo, progress: stepProgress(state) };
}

type StepNavModel = ReturnType<typeof useStepNav>;

function StepTile({ step, className }: { step: StepView; className?: string }) {
  const Icon = step.status === "done" ? IconCheckOutline18 : step.icon;
  return (
    <span
      data-status={step.status}
      className={cn(
        "flex size-8 shrink-0 items-center justify-center rounded-sm border border-grayA-4 text-gray-11 data-[status=done]:border-gray-12 data-[status=done]:bg-gray-12 data-[status=done]:text-gray-1",
        className,
      )}
    >
      <Icon className="size-4" />
    </span>
  );
}

const DETAIL_ENTER = { duration: 0.22, ease: [0.215, 0.61, 0.355, 1] } as const;
const HIGHLIGHT_MOVE = { type: "spring", duration: 0.4, bounce: 0 } as const;
const ROW_EXIT = { duration: 0.3, ease: [0.645, 0.045, 0.355, 1] } as const;

function StepItem({ children, className }: { children: ReactNode; className: string }) {
  const reduceMotion = useReducedMotion();
  return (
    <m.li
      exit={{ opacity: 0, height: 0, paddingBottom: 0 }}
      transition={reduceMotion ? { duration: 0 } : ROW_EXIT}
      className={cn("overflow-hidden", className)}
    >
      {children}
    </m.li>
  );
}

function StepHighlight() {
  const reduceMotion = useReducedMotion();
  return (
    <m.span
      layoutId="step-highlight"
      transition={reduceMotion ? { duration: 0 } : HIGHLIGHT_MOVE}
      className="absolute inset-0 -z-10 rounded-md bg-raised shadow-xs ring-1 ring-grayA-4"
    />
  );
}

function StepDetail({ detail }: { detail: string | null }) {
  const reduceMotion = useReducedMotion();
  return (
    <AnimatePresence initial={false}>
      {detail ? (
        <m.span
          key="detail"
          initial={{ opacity: 0, y: -4 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0 }}
          transition={reduceMotion ? { duration: 0 } : DETAIL_ENTER}
          className="truncate text-xs text-gray-11"
        >
          {detail}
        </m.span>
      ) : null}
    </AnimatePresence>
  );
}

function StepListRow({ step, onSelect }: { step: StepView; onSelect: () => void }) {
  const reduceMotion = useReducedMotion();
  const editable = step.status === "done";
  return (
    <button
      type="button"
      disabled={!editable}
      aria-current={step.status === "current" ? "step" : undefined}
      onClick={onSelect}
      data-status={step.status}
      className="group relative isolate flex h-14 w-full items-center gap-3 rounded-md border border-transparent px-3 text-left transition-colors enabled:cursor-pointer enabled:hover:bg-grayA-3 disabled:cursor-default"
    >
      {step.status === "current" ? <StepHighlight /> : null}
      <StepTile step={step} />
      <m.span
        layout="position"
        transition={reduceMotion ? { duration: 0 } : DETAIL_ENTER}
        className="flex min-w-0 flex-col"
      >
        <span className="text-sm font-medium text-gray-10 group-data-[status=current]:text-gray-12 group-data-[status=done]:text-gray-11">
          {step.title}
        </span>
        <StepDetail detail={step.detail} />
      </m.span>
      {editable ? (
        <Button
          variant="ghost"
          render={<span />}
          className="ml-auto size-7 shrink-0 p-0 text-gray-9 group-hover:bg-grayA-3 group-hover:text-gray-11"
        >
          <IconPenWriting3Outline18 aria-hidden className="size-4" />
          <span className="sr-only">Edit</span>
        </Button>
      ) : null}
    </button>
  );
}

function StepList({
  nav: { steps, goTo },
  onSelect,
}: {
  nav: StepNavModel;
  onSelect?: () => void;
}) {
  const groupId = useId();
  return (
    <LayoutGroup id={groupId}>
      <ol className="flex flex-col">
        <AnimatePresence initial={false}>
          {steps.map((step) => (
            <StepItem key={step.id} className="pb-1">
              <StepListRow
                step={step}
                onSelect={() => {
                  goTo(step);
                  onSelect?.();
                }}
              />
            </StepItem>
          ))}
        </AnimatePresence>
      </ol>
    </LayoutGroup>
  );
}

function StepRail({ nav: { steps, goTo } }: { nav: StepNavModel }) {
  return (
    <ol className="flex flex-col items-center">
      <AnimatePresence initial={false}>
        {steps.map((step) => (
          <StepItem key={step.id} className="pb-2">
            <Tooltip>
              <TooltipTrigger
                render={
                  <button
                    type="button"
                    aria-label={step.detail ? `${step.title}: ${step.detail}` : step.title}
                    aria-current={step.status === "current" ? "step" : undefined}
                    aria-disabled={step.status !== "done"}
                    onClick={() => goTo(step)}
                    className="flex rounded-lg p-1 transition-colors aria-disabled:cursor-default not-aria-disabled:cursor-pointer not-aria-disabled:hover:bg-grayA-3"
                  />
                }
              >
                <StepTile
                  step={step}
                  className="data-[status=current]:border-gray-9 data-[status=current]:ring-3 data-[status=current]:ring-gray-4"
                />
              </TooltipTrigger>
              <TooltipContent side="right" className="flex flex-col">
                <span>{step.title}</span>
                {step.detail ? <span className="font-normal opacity-60">{step.detail}</span> : null}
              </TooltipContent>
            </Tooltip>
          </StepItem>
        ))}
      </AnimatePresence>
    </ol>
  );
}

function StepStrip({ nav }: { nav: StepNavModel }) {
  const {
    steps,
    progress: { at, label },
  } = nav;
  const [open, setOpen] = useState(false);
  return (
    <Drawer.Root open={open} onOpenChange={setOpen}>
      <Drawer.Trigger className="flex h-10 shrink-0 items-center gap-2.5 border-b border-grayA-4 px-4 text-xs text-gray-11 lg:hidden">
        <span>
          {label} · <span className="font-medium text-gray-12">{steps[at]?.title}</span>
        </span>
        <span className="ml-auto flex gap-0.75">
          {steps.map((step, index) => (
            <i
              key={step.id}
              className={cn("h-0.75 w-3.5 rounded-full bg-gray-5", index <= at && "bg-gray-12")}
            />
          ))}
        </span>
      </Drawer.Trigger>
      <Drawer.Content className="px-4 pt-4 pb-6">
        <Drawer.Title className="sr-only">Steps</Drawer.Title>
        <Drawer.Description className="sr-only">Go back to a finished step.</Drawer.Description>
        <StepList nav={nav} onSelect={() => setOpen(false)} />
      </Drawer.Content>
    </Drawer.Root>
  );
}

export function StepNav() {
  const nav = useStepNav();
  return (
    <>
      <StepStrip nav={nav} />
      <StepAside>
        <div className="hidden xl:block">
          <StepList nav={nav} />
        </div>
        <div className="xl:hidden">
          <StepRail nav={nav} />
        </div>
      </StepAside>
    </>
  );
}
