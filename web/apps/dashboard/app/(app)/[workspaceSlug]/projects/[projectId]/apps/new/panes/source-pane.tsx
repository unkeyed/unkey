"use client";

import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { trpc } from "@/lib/trpc/client";
import { Github, IconChevronRightOutline18, IconCubeOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import type { ReactNode } from "react";
import { useNewAppFlow } from "../flow";
import { CONNECT_GITHUB, useConnectGithub } from "../use-connect-github";
import { type SourceKind, sourceCopy } from "../wizard-model";
import { PaneRow } from "./pane-row";
import { type SourceAction, sourceAction } from "./source/source-choice-state";

const sourceIcon: Record<SourceKind, ReactNode> = {
  git: <Github />,
  oci: <IconCubeOutline18 />,
};

const SOURCE_KINDS: readonly SourceKind[] = ["git", "oci"];

const SOURCE_ROW: Record<SourceAction, (kind: SourceKind) => { title: string; hint: string }> = {
  pick: (kind) => ({ title: sourceCopy[kind].sourceLabel, hint: sourceCopy[kind].hint }),
  "connect-github": () => CONNECT_GITHUB,
};

export function SourcePane({ selected }: { selected: SourceKind | null }) {
  const { locked, dispatch } = useNewAppFlow();
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const github = useConnectGithub();
  const { data: context } = trpc.deploy.project.creationContext.useQuery();
  const needsGithub = context?.hasGithubInstallation === false;

  const choose = (kind: SourceKind, action: SourceAction) => {
    if (gated) {
      openPaywall();
      return;
    }
    match(action)
      .with("connect-github", () => github.connect())
      .with("pick", () => dispatch({ type: "pick-source", source: kind }))
      .exhaustive();
  };

  return (
    <>
      <div className="flex flex-col gap-2">
        {SOURCE_KINDS.map((kind) => {
          const action = sourceAction(kind, needsGithub);
          const row = SOURCE_ROW[action](kind);
          return (
            <PaneRow
              key={kind}
              variant="outline"
              icon={sourceIcon[kind]}
              label={row.title}
              hint={row.hint}
              trailing={<IconChevronRightOutline18 />}
              selected={selected === kind}
              disabled={locked}
              onClick={() => choose(kind, action)}
            />
          );
        })}
      </div>
      {github.error ? (
        <p role="alert" className="text-sm text-error-11">
          {github.error}
        </p>
      ) : null}
      {planGate}
    </>
  );
}
