"use client";

import { SettingField } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/[environmentSlug]/settings/components/shared/form-blocks";
import {
  FormSettingCard,
  resolveSaveState,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/[environmentSlug]/settings/components/shared/form-setting-card";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { type Project, createProjectRequestSchema } from "@/lib/collections/deploy/projects";
import { zodResolver } from "@hookform/resolvers/zod";
import { FormInput, SettingCardGroup, SettingsGroup, SettingsRow } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import type { z } from "zod";

const nameSchema = createProjectRequestSchema.pick({ name: true });

export function UpdateProjectSettings({ project }: { project: Project }) {
  const workspace = useWorkspaceNavigation();

  if (project.isDefault) {
    return (
      <SettingCardGroup>
        <SettingsRow
          title="Project name"
          description="The default project is named after your workspace. Rename it in workspace settings."
        >
          <span className="text-sm text-gray-12">{workspace.name}</span>
        </SettingsRow>
      </SettingCardGroup>
    );
  }

  return (
    <SettingsGroup>
      <ProjectNameCard project={project} />
    </SettingsGroup>
  );
}

function ProjectNameCard({ project }: { project: Project }) {
  const {
    register,
    handleSubmit,
    control,
    formState: { isValid, isSubmitting, errors },
  } = useForm<z.infer<typeof nameSchema>>({
    resolver: zodResolver(nameSchema),
    mode: "onChange",
    defaultValues: { name: project.name },
  });

  const current = useWatch({ control, name: "name" });
  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [current === project.name, { status: "disabled", reason: "No changes to save" }],
  ]);

  const onSubmit = async (values: z.infer<typeof nameSchema>) => {
    const tx = collection.projects.update(project.id, (draft) => {
      draft.name = values.name;
    });
    await tx.isPersisted.promise;
  };

  return (
    <FormSettingCard
      title="Project name"
      description="A descriptive name for your project."
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
    >
      <SettingField>
        <FormInput
          label="Project name"
          requirement="required"
          placeholder="My Awesome Project"
          error={errors.name?.message}
          {...register("name")}
        />
      </SettingField>
    </FormSettingCard>
  );
}
