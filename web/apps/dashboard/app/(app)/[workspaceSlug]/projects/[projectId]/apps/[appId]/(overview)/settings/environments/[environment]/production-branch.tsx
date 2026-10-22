"use client";

import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import {
  SettingsForm,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import type { ReactNode } from "react";
import { z } from "zod";
import {
  BranchCombobox,
  type BranchConnection,
} from "../../../../../_components/branches/branch-combobox";
import { ComboboxSkeleton } from "../../../../../_components/settings/build-settings/github-settings/shared";
import { useBuildSource } from "../../../../../_components/settings/hooks/use-build-source";
import { useSavableForm } from "../../../../../_components/settings/hooks/use-setting-form";
import { BranchLabel } from "../branch-label";

const branchSchema = z.object({ branch: z.string().min(1, "Select a branch") });

export function ProductionBranch() {
  const { projectId, appId, app, connection, installations } = useBuildSource();
  if (app && connection) {
    return (
      <ProductionBranchForm
        projectId={projectId}
        appId={appId}
        branch={app.defaultBranch}
        connection={connection}
      />
    );
  }
  return (
    <BranchRow>
      {installations.isError ? (
        <BranchLabel branch={{ kind: "none" }} className="w-fit" />
      ) : (
        <div className="w-full max-w-(--setting-w)">
          <ComboboxSkeleton />
        </div>
      )}
    </BranchRow>
  );
}

function ProductionBranchForm({
  projectId,
  appId,
  branch,
  connection,
}: { projectId: string; appId: string; branch: string; connection: BranchConnection }) {
  const utils = trpc.useUtils();
  const { form, formProps } = useSavableForm({
    schema: branchSchema,
    values: { branch },
    save: async (values) => {
      await collection.apps.update(appId, { optimistic: false }, (draft) => {
        draft.defaultBranch = values.branch;
      }).isPersisted.promise;
      await Promise.all([
        utils.github.getInstallations.invalidate({ projectId, appId }),
        utils.github.getRepoTree.invalidate({ projectId, appId }),
      ]);
    },
  });

  return (
    <SettingsForm {...formProps}>
      <BranchRow>
        <BranchCombobox
          aria-label="Production branch"
          className="max-w-(--setting-w)"
          projectId={projectId}
          connection={connection}
          value={form.watch("branch")}
          onSelect={(selected) =>
            form.setValue("branch", selected, { shouldDirty: true, shouldValidate: true })
          }
        />
      </BranchRow>
    </SettingsForm>
  );
}

function BranchRow({ children }: { children: ReactNode }) {
  return (
    <SettingsRow>
      <SettingsRowHeader>
        <SettingsRowTitle>Branch</SettingsRowTitle>
        <SettingsRowDescription>Pushes to this branch deploy to production.</SettingsRowDescription>
      </SettingsRowHeader>
      <SettingsRowContent>{children}</SettingsRowContent>
    </SettingsRow>
  );
}
