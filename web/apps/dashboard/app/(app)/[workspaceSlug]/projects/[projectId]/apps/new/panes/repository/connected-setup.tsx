"use client";

import { collection } from "@/lib/collections";
import { githubUrl } from "@/lib/github-url";
import { slugify } from "@/lib/slugify";
import { getErrorMessage } from "@/lib/unkey-client";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { Github, IconArrowUpRightOutline12 } from "@unkey/icons";
import { FormInput, toast } from "@unkey/ui";
import { useState } from "react";
import type { SetupFieldFocus } from "../../wizard-model";
import { useAppSettings } from "../settings";
import { Field, SettingsForm } from "../settings/settings-form";
import { SettingsFormSkeleton } from "../settings/skeleton";
import { VariablesSection, useVariableDraft } from "../variables/variables-section";
import { BranchField } from "./branch-field";
import type { Connection } from "./repository-view";

type ConnectedSetupProps = {
  projectId: string;
  appId: string;
  connection: Connection;
  branchDisabled: boolean;
  onBranchChange: (branch: string) => void;
  onRename: (name: string) => Promise<void>;
  onContinue: () => void;
  focusField: SetupFieldFocus | null;
};

export function ConnectedSetup({
  projectId,
  appId,
  connection,
  branchDisabled,
  onBranchChange,
  onRename,
  onContinue,
  focusField,
}: ConnectedSetupProps) {
  const variables = useVariableDraft(projectId, appId);
  const settings = useAppSettings(projectId, appId);
  const { data: apps } = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  const savedName = apps.at(0)?.name ?? "";
  const [draftName, setDraftName] = useState<string | null>(null);
  const name = draftName ?? savedName;
  const repoHref = githubUrl.repo(connection.repositoryFullName);

  const saveAndContinue = async () => {
    const slug = slugify(name);
    if (slug && slug !== savedName) {
      try {
        await onRename(slug);
      } catch (error) {
        toast.error("Could not rename the app", { description: getErrorMessage(error) });
        return;
      }
    }
    if (!(await variables.save())) {
      return;
    }
    onContinue();
  };

  return (
    <div className="flex flex-1 flex-col gap-4">
      <a
        href={repoHref}
        target="_blank"
        rel="noopener noreferrer"
        className="flex w-full min-w-0 items-center gap-2 rounded-md border border-grayA-4 bg-grayA-2 px-3 py-2 text-sm font-medium text-gray-12 hover:bg-grayA-3"
      >
        <Github className="size-3.5 shrink-0" />
        <span className="min-w-0 flex-1 truncate">{connection.repositoryFullName}</span>
        <IconArrowUpRightOutline12 className="size-3 shrink-0 text-gray-10" />
      </a>
      {settings.status === "loading" ? (
        <SettingsFormSkeleton />
      ) : (
        <SettingsForm
          projectId={projectId}
          appId={appId}
          source="git"
          production={settings.production}
          environmentIds={settings.environmentIds}
          onSaved={saveAndContinue}
          focusField={focusField}
          pairedField={{
            title: "App name",
            control: (
              <FormInput
                aria-label="App name"
                data-1p-ignore
                autoComplete="off"
                value={name}
                onChange={(e) => setDraftName(e.target.value)}
              />
            ),
          }}
          sections={[
            {
              title: "Environment variables",
              hint:
                variables.existingKeys.length > 0
                  ? `${variables.existingKeys.length} set`
                  : "Optional",
              content: <VariablesSection draft={variables} />,
            },
          ]}
          advancedLeadingRows={
            <Field title="Branch">
              <BranchField
                projectId={projectId}
                installationId={connection.installationId}
                repositoryFullName={connection.repositoryFullName}
                branch={connection.branch}
                disabled={branchDisabled}
                onChange={onBranchChange}
              />
            </Field>
          }
        />
      )}
    </div>
  );
}
