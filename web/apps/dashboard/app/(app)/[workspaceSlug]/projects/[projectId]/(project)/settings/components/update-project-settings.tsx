"use client";

import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { SettingField } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/settings/components/shared/form-blocks";
import {
  FormSettingCard,
  resolveSaveState,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/settings/components/shared/form-setting-card";
import { collection } from "@/lib/collections";
import { type Project, createProjectRequestSchema } from "@/lib/collections/deploy/projects";
import { zodResolver } from "@hookform/resolvers/zod";
import { Cube } from "@unkey/icons";
import { FormInput, SettingCardGroup } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import type { z } from "zod";

const nameSchema = createProjectRequestSchema.pick({ name: true });

export function UpdateProjectSettings() {
  const { project } = useProjectData();

  if (!project) {
    return null;
  }

  return (
    <SettingCardGroup>
      <ProjectNameCard project={project} />
    </SettingCardGroup>
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
      icon={<Cube className="text-gray-12" iconSize="xl-medium" />}
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
