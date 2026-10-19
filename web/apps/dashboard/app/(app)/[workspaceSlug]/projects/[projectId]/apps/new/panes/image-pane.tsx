"use client";

import { useApp } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/hooks/use-app";
import { OCIImage } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/oci-image";
import { ociImageReferenceSchema } from "@/lib/collections/deploy/apps";
import { sanitizeImageRef, validateImageRef } from "@/lib/docker-image-ref";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import {
  FormInput,
  SettingsGroup,
  SettingsGroupContent,
  Skeleton,
  UnsavedChangesScope,
  useUnsavedChanges,
} from "@unkey/ui";
import { useId } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { appNameFromImage } from "../app-name";
import { NEW_APP_MUTATION_KEY, useNewAppFlow } from "../flow";
import { newAppErrorMessage, useAppLifecycle } from "../use-app-lifecycle";
import { PaneSubmit } from "./pane-actions";

export function ImagePane({ appId }: { appId: string | null }) {
  return appId ? <SavedImage appId={appId} /> : <NewImage />;
}

const imageSchema = z.object({ image: ociImageReferenceSchema });

function imageWarning(image: string): string | undefined {
  const validation = validateImageRef(sanitizeImageRef(image));
  return validation.ok ? validation.warning : undefined;
}

function NewImage() {
  const { projectId, dispatch, ensureApp } = useNewAppFlow();
  const { createImageApp } = useAppLifecycle(projectId);
  const create = useMutation({
    mutationKey: NEW_APP_MUTATION_KEY,
    mutationFn: (image: string) =>
      ensureApp("oci", () => createImageApp(appNameFromImage(image), image)),
    onSuccess: () => dispatch({ type: "next" }),
  });
  const {
    register,
    handleSubmit,
    control,
    formState: { errors },
  } = useForm<z.input<typeof imageSchema>, unknown, z.output<typeof imageSchema>>({
    resolver: zodResolver(imageSchema),
    defaultValues: { image: "" },
  });
  const warning = imageWarning(useWatch({ control, name: "image" }));

  const onValid = ({ image }: z.output<typeof imageSchema>) => {
    if (!create.isLoading) {
      create.mutate(image);
    }
  };

  const formId = useId();
  return (
    <form id={formId} className="flex flex-col gap-3" onSubmit={handleSubmit(onValid)}>
      <FormInput
        label="Image"
        autoFocus
        spellCheck={false}
        data-1p-ignore
        autoComplete="off"
        placeholder="ghcr.io/acme/api:v1.4.2"
        className="font-mono"
        error={
          errors.image?.message ?? (create.error ? newAppErrorMessage(create.error) : undefined)
        }
        variant={warning ? "warning" : "default"}
        description={warning}
        {...register("image")}
      />
      <PaneSubmit form={formId} loading={create.isLoading} />
    </form>
  );
}

function SavedImage({ appId }: { appId: string }) {
  const { dispatch } = useNewAppFlow();
  const { isDirty, report } = useUnsavedChanges();
  const { app } = useApp();
  if (!app) {
    return <Skeleton className="h-24 w-full rounded-lg" />;
  }
  return (
    <UnsavedChangesScope report={report}>
      <SettingsGroup>
        <SettingsGroupContent>
          <OCIImage appId={appId} imageReference={app.imageReference ?? ""} />
        </SettingsGroupContent>
      </SettingsGroup>
      <PaneSubmit
        loading={false}
        disabled={isDirty}
        notice={
          isDirty ? (
            <span className="text-xs text-gray-11">Save your changes to continue</span>
          ) : null
        }
        onClick={() => dispatch({ type: "next" })}
      />
    </UnsavedChangesScope>
  );
}
