"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { type Project, createProjectRequestSchema } from "@/lib/collections/deploy/projects";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  FormInput,
  SettingsForm,
  SettingsGroup,
  SettingsGroupContent,
  SettingsRow,
  formSaveState,
} from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import type { z } from "zod";

const nameSchema = createProjectRequestSchema.pick({ name: true });

export function UpdateProjectSettings({ project }: { project: Project }) {
  const workspace = useWorkspaceNavigation();

  if (project.isDefault) {
    return (
      <SettingsGroup>
        <SettingsGroupContent>
          <SettingsRow
            title="Project name"
            description="The default project is named after your workspace. Rename it in workspace settings."
          >
            <span className="text-sm text-gray-12">{workspace.name}</span>
          </SettingsRow>
        </SettingsGroupContent>
      </SettingsGroup>
    );
  }

  return (
    <SettingsGroup>
      <SettingsGroupContent>
        <ProjectNameRow project={project} />
      </SettingsGroupContent>
    </SettingsGroup>
  );
}

function ProjectNameRow({ project }: { project: Project }) {
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

  const isDirty = useWatch({ control, name: "name" }).trim() !== project.name.trim();
  const saveState = formSaveState({ isSubmitting, isValid, isDirty });

  const onSubmit = async (values: z.infer<typeof nameSchema>) => {
    const tx = collection.projects.update(project.id, (draft) => {
      draft.name = values.name;
    });
    await tx.isPersisted.promise;
  };

  return (
    <SettingsForm dirty={isDirty} onSubmit={handleSubmit(onSubmit)} saveState={saveState}>
      <SettingsRow title="Project name" description="A descriptive name for your project.">
        <FormInput
          aria-label="Project name"
          requirement="required"
          placeholder="My Awesome Project"
          error={errors.name?.message}
          {...register("name")}
        />
      </SettingsRow>
    </SettingsForm>
  );
}
