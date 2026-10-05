"use client";

import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { trpc } from "@/lib/trpc/client";
import { Github, IconChevronRightOutline18, IconCubeOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import { Item, ItemActions, ItemContent, ItemDescription, ItemMedia, ItemTitle } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import type { ReactNode } from "react";
import { useNewAppFlow } from "../flow";
import { useConnectGithub } from "../use-connect-github";
import type { SourceKind } from "../wizard-model";
import { type SourceChoice, sourceChoices } from "./source/source-choice-state";

const sourceIcon: Record<SourceKind, ReactNode> = {
  git: <Github />,
  oci: <IconCubeOutline18 />,
};

export function SourcePane({
  selected,
  lockedTo,
}: {
  selected: SourceKind | null;
  lockedTo: SourceKind | null;
}) {
  const { state, dispatch } = useNewAppFlow();
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const github = useConnectGithub();
  const { data: context } = trpc.deploy.project.creationContext.useQuery();
  const needsGithub = context?.hasGithubInstallation === false;

  const choose = (choice: SourceChoice) => {
    if (gated) {
      openPaywall();
      return;
    }
    match(choice.action)
      .with("connect-github", () => github.connect())
      .with("pick", () => dispatch({ type: "pick-source", source: choice.kind }))
      .with("locked", () => undefined)
      .exhaustive();
  };

  return (
    <>
      <div className="flex flex-col gap-2">
        {sourceChoices({ lockedTo, needsGithub }).map((choice) => {
          const { kind, title, description } = choice;
          return (
            <Item
              key={kind}
              variant="outline"
              className={cn(
                "bg-raised disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:bg-raised",
                selected === kind && "border-grayA-8 bg-grayA-2",
              )}
              render={
                <button
                  type="button"
                  disabled={state.pending || github.connecting || choice.action === "locked"}
                  aria-pressed={selected === kind}
                  onClick={() => choose(choice)}
                >
                  <ItemMedia>{sourceIcon[kind]}</ItemMedia>
                  <ItemContent>
                    <ItemTitle>{title}</ItemTitle>
                    <ItemDescription>{description}</ItemDescription>
                  </ItemContent>
                  <ItemActions>
                    <IconChevronRightOutline18 />
                  </ItemActions>
                </button>
              }
            />
          );
        })}
      </div>
      {github.error ? <p className="text-sm text-error-11">{github.error}</p> : null}
      {planGate}
    </>
  );
}
