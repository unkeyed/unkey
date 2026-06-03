"use client";

import { collection } from "@/lib/collections";
import { type Project, createProjectRequestSchema } from "@/lib/collections/deploy/projects";
import { zodResolver } from "@hookform/resolvers/zod";
import { Cube, Tag } from "@unkey/icons";
import { FormInput, SettingCardGroup } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import type { z } from "zod";
import { useProjectData } from "../../../apps/[appId]/(overview)/data-provider";
import { SettingField } from "../../../apps/[appId]/(overview)/settings/components/shared/form-blocks";
import {
  FormSettingCard,
  resolveSaveState,
} from "../../../apps/[appId]/(overview)/settings/components/shared/form-setting-card";

const nameSchema = createProjectRequestSchema.pick({ name: true });
const slugSchema = createProjectRequestSchema.pick({ slug: true });

export function UpdateProjectSettings() {
  const { project } = useProjectData();

  if (!project) {
    return null;
  }

  return (
    <SettingCardGroup>
      <ProjectNameCard project={project} />
      <ProjectSlugCard project={project} />
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
    collection.projects.update(project.id, (draft) => {
      draft.name = values.name;
    });
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

function ProjectSlugCard({ project }: { project: Project }) {
  const {
    register,
    handleSubmit,
    control,
    formState: { isValid, isSubmitting, errors },
  } = useForm<z.infer<typeof slugSchema>>({
    resolver: zodResolver(slugSchema),
    mode: "onChange",
    defaultValues: { slug: project.slug },
  });

  const current = useWatch({ control, name: "slug" });
  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [current === project.slug, { status: "disabled", reason: "No changes to save" }],
  ]);

  const onSubmit = async (values: z.infer<typeof slugSchema>) => {
    collection.projects.update(project.id, (draft) => {
      draft.slug = values.slug;
    });
  };

  return (
    <FormSettingCard
      icon={<Tag className="text-gray-12" iconSize="xl-medium" />}
      title="Project slug"
      description="URL-friendly identifier for your project, unique within the workspace."
      displayValue={project.slug}
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
    >
      <SettingField>
        <FormInput
          label="Project slug"
          requirement="required"
          placeholder="my-awesome-project"
          error={errors.slug?.message}
          {...register("slug")}
        />
      </SettingField>
    </FormSettingCard>
  );
}
