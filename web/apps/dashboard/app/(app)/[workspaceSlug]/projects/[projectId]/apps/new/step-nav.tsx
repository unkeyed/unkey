"use client";

import { Presence } from "@/components/presence";
import { SPRING_SETTLE, useLayoutMotion } from "@/hooks/use-layout-motion";
import { collection } from "@/lib/collections";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
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
import {
  type ComponentType,
  type ReactNode,
  type RefObject,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
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
  const appId = state.appId ?? "";
  const { data } = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  const app = data.at(0);
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

const DETAIL_EASE = "cubic-bezier(0.215, 0.61, 0.355, 1)";
const DETAIL_MOVE = { duration: 220, easing: DETAIL_EASE };
const ROW_EXIT = { duration: 300, easing: "cubic-bezier(0.645, 0.045, 0.355, 1)" };

function useRowExit(list: RefObject<HTMLOListElement | null>) {
  useLayoutMotion(list, {
    appear: false,
    move: ROW_EXIT,
    exit: () => ({ keyframes: [{ opacity: 0 }], timing: ROW_EXIT }),
  });
}

function StepItem({
  id,
  children,
  className,
}: { id: Step; children: ReactNode; className: string }) {
  return (
    <li data-motion-key={id} className={cn("overflow-hidden", className)}>
      {children}
    </li>
  );
}

function StepHighlight({ list }: { list: RefObject<HTMLOListElement | null> }) {
  const [box, setBox] = useState<{ top: number; height: number } | null>(null);
  useLayoutEffect(() => {
    const row = list.current?.querySelector<HTMLElement>('[aria-current="step"]');
    const next = row ? { top: row.offsetTop, height: row.offsetHeight } : null;
    setBox((prev) => (prev?.top === next?.top && prev?.height === next?.height ? prev : next));
  });
  if (!box) {
    return null;
  }
  return (
    <span
      aria-hidden
      className="pointer-events-none absolute inset-x-0 top-0 -z-10 rounded-md bg-raised shadow-xs ring-1 ring-grayA-4 transition-[translate] duration-400 motion-reduce:transition-none"
      style={{
        height: box.height,
        translate: `0 ${box.top}px`,
        transitionTimingFunction: SPRING_SETTLE,
      }}
    />
  );
}

function StepDetail({ detail }: { detail: string | null }) {
  return (
    <Presence present={detail !== null} exitMs={220}>
      {({ closing, presentOnMount }) => (
        <span
          className={cn(
            "truncate text-xs text-gray-11 transition-[opacity,translate] duration-[220ms] motion-reduce:transition-none",
            closing && "opacity-0",
            !closing && !presentOnMount && "starting:opacity-0 starting:-translate-y-1",
          )}
          style={{ transitionTimingFunction: DETAIL_EASE }}
        >
          {detail}
        </span>
      )}
    </Presence>
  );
}

function StepListRow({ step, onSelect }: { step: StepView; onSelect: () => void }) {
  const rowRef = useRef<HTMLButtonElement>(null);
  useLayoutMotion(rowRef, { appear: false, move: DETAIL_MOVE });
  const editable = step.status === "done";
  return (
    <button
      ref={rowRef}
      type="button"
      disabled={!editable}
      aria-current={step.status === "current" ? "step" : undefined}
      onClick={onSelect}
      data-status={step.status}
      className="group relative isolate flex h-14 w-full items-center gap-3 rounded-md border border-transparent px-3 text-left transition-colors enabled:cursor-pointer enabled:hover:bg-grayA-3 disabled:cursor-default"
    >
      <StepTile step={step} />
      <span data-motion-key="title" className="flex min-w-0 flex-col">
        <span className="text-sm font-medium text-gray-10 group-data-[status=current]:text-gray-12 group-data-[status=done]:text-gray-11">
          {step.title}
        </span>
        <StepDetail detail={step.detail} />
      </span>
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
  const listRef = useRef<HTMLOListElement>(null);
  useRowExit(listRef);
  return (
    <ol ref={listRef} className="relative isolate flex flex-col">
      <StepHighlight list={listRef} />
      {steps.map((step) => (
        <StepItem key={step.id} id={step.id} className="pb-1">
          <StepListRow
            step={step}
            onSelect={() => {
              goTo(step);
              onSelect?.();
            }}
          />
        </StepItem>
      ))}
    </ol>
  );
}

function StepRail({ nav: { steps, goTo } }: { nav: StepNavModel }) {
  const listRef = useRef<HTMLOListElement>(null);
  useRowExit(listRef);
  return (
    <ol ref={listRef} className="relative flex flex-col items-center">
      {steps.map((step) => (
        <StepItem key={step.id} id={step.id} className="pb-2">
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
