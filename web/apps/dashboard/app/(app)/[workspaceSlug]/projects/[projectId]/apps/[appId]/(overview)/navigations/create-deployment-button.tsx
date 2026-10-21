"use client";

import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { collection } from "@/lib/collections";
import type { App } from "@/lib/collections/deploy/apps";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { IconPlusOutline12 } from "@unkey/icons";
import { match } from "@unkey/match";
import type React from "react";
import { useState } from "react";
import { useAppId, useProjectData } from "../data-provider";
import { GitDeploymentForm } from "./git-deployment-form";
import { ImageDeploymentForm } from "./image-deployment-form";

const DEPLOYMENT_FORMS = {
  git: GitDeploymentForm,
  oci: ImageDeploymentForm,
} satisfies Record<"git" | "oci", typeof GitDeploymentForm>;

function deploymentSource(app: App): keyof typeof DEPLOYMENT_FORMS {
  return match(app.sourceType)
    .returnType<keyof typeof DEPLOYMENT_FORMS>()
    .with("git", () => "git")
    .with("oci", () => "oci")
    .with("unknown", () => (app.repositoryFullName ? "git" : "oci"))
    .exhaustive();
}

type Props = {
  defaultOpen?: boolean;
  renderTrigger?: (props: { onClick: () => void }) => React.ReactNode;
};

export function CreateDeploymentButton({
  defaultOpen,
  renderTrigger,
  ...rest
}: React.ButtonHTMLAttributes<HTMLButtonElement> & Props) {
  const [isOpen, setIsOpen] = useState(defaultOpen ?? false);
  const { projectId } = useProjectData();
  const appId = useAppId();
  const { gated, openPaywall, planGate } = useDeployActionGate();

  // Without a Compute plan the trigger opens the paywall instead of the create
  // dialog; the backend still rejects the mutation either way.
  const openDeploymentDialog = () => (gated ? openPaywall() : setIsOpen(true));

  // Repo connections are per-app, not per-project; the project-level
  // repositoryFullName is just some app's connection in this project.
  const appQuery = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  const app = appQuery.data?.[0];
  const Form = app ? DEPLOYMENT_FORMS[deploymentSource(app)] : null;

  return (
    <>
      {renderTrigger ? (
        renderTrigger({ onClick: openDeploymentDialog })
      ) : (
        <NavbarActionButton
          {...rest}
          color="default"
          variant="outline"
          className="size-7"
          onClick={openDeploymentDialog}
        >
          <IconPlusOutline12 />
        </NavbarActionButton>
      )}
      {planGate}
      {Form && app && <Form app={app} isOpen={isOpen} onOpenChange={setIsOpen} />}
    </>
  );
}
