"use client";

import { SettingField } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/settings/components/shared/form-blocks";
import {
  FormSettingCard,
  resolveSaveState,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/settings/components/shared/form-setting-card";
import { SelectedConfig } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/settings/components/shared/selected-config";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { type Project, createProjectRequestSchema } from "@/lib/collections/deploy/projects";
import { zodResolver } from "@hookform/resolvers/zod";
import { IconCubeOutline18 } from "@unkey/icons";
import { FormInput, SettingCard, SettingCardGroup } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import type { z } from "zod";

const nameSchema = createProjectRequestSchema.pick({ name: true });

export function UpdateProjectSettings({ project }: { project: Project }) {
  const workspace = useWorkspaceNavigation();

  return (
    <SettingCardGroup>
      {project.isDefault ? (
        <WorkspaceNameCard name={workspace.name} />
      ) : (
        <ProjectNameCard project={project} />
      )}
    </SettingCardGroup>
  );
}

function WorkspaceNameCard({ name }: { name: string }) {
  return (
    <SettingCard
      className="px-4 py-[18px]"
      icon={<IconCubeOutline18 className="text-gray-12" />}
      title="Project name"
      description="The default project is named after your workspace. Rename it in workspace settings."
      contentWidth="w-full lg:w-[320px] justify-end"
    >
      <SelectedConfig label={name} />
    </SettingCard>
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
      icon={<IconCubeOutline18 className="text-gray-12" />}
      title="Project name"
      description="A descriptive name for your project."
      displayValue={project.name}
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
