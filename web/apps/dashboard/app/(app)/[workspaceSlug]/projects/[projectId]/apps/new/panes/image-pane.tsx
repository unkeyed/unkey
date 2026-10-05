"use client";

import { collection } from "@/lib/collections";
import { ENVIRONMENT_SETTINGS_DEFAULTS } from "@/lib/collections/deploy/environment-settings";
import { sanitizeImageRef, validateImageRef } from "@/lib/docker-image-ref";
import { trpc } from "@/lib/trpc/client";
import { getErrorMessage } from "@/lib/unkey-client";
import { zodResolver } from "@hookform/resolvers/zod";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { IconCubeOutline18 } from "@unkey/icons";
import { Button, FormInput, toast } from "@unkey/ui";
import { useEffect, useId, useState } from "react";
import { type UseFormRegisterReturn, useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { appNameFromImage } from "../app-name";
import { useNewAppFlow } from "../flow";
import { useAppLifecycle } from "../use-app-lifecycle";
import { PaneActions } from "./pane-actions";
import { AppSettingsForm } from "./settings";
import { deploymentConfigSchema, settingTitle } from "./settings/deployment-config";
import { RegionSelect } from "./settings/region-select";
import { Field, FieldStack } from "./settings/settings-form";
import { SizeField } from "./settings/size-field";

export function ImagePane({ appId }: { appId: string | null }) {
  const { projectId, dispatch } = useNewAppFlow();
  if (appId) {
    return (
      <SavedImage
        projectId={projectId}
        appId={appId}
        onContinue={() => dispatch({ type: "next" })}
      />
    );
  }
  return <NewImage />;
}

const imageSchema = z
  .string()
  .transform(sanitizeImageRef)
  .superRefine((ref, ctx) => {
    const validation = validateImageRef(ref);
    if (!validation.ok) {
      ctx.addIssue({
        code: "custom",
        message: ref ? validation.error : "Enter an image, like ghcr.io/acme/api:1.0",
      });
    }
  });

function imageWarning(image: string): string | undefined {
  const validation = validateImageRef(sanitizeImageRef(image));
  return validation.ok ? validation.warning : undefined;
}

function ImageInput({
  registration,
  value,
  error,
  autoFocus,
}: {
  registration: UseFormRegisterReturn;
  value: string;
  error: string | undefined;
  autoFocus?: boolean;
}) {
  const warning = imageWarning(value);
  return (
    <FormInput
      aria-label="Image"
      autoFocus={autoFocus}
      spellCheck={false}
      data-1p-ignore
      autoComplete="off"
      placeholder="ghcr.io/acme/api:v1.4.2"
      className="font-mono"
      error={error}
      variant={warning ? "warning" : "default"}
      description={warning}
      {...registration}
    />
  );
}

const newImageSchema = deploymentConfigSchema
  .pick({ port: true, regions: true, size: true })
  .extend({ image: imageSchema });

type NewImageValues = z.input<typeof newImageSchema>;
type NewImageOutput = z.output<typeof newImageSchema>;

function NewImage() {
  const { projectId, dispatch, ensureApp } = useNewAppFlow();
  const { createImageApp } = useAppLifecycle(projectId);
  const { data: availableRegions } = trpc.deploy.environmentSettings.getAvailableRegions.useQuery();
  const {
    register,
    handleSubmit,
    setValue,
    control,
    getValues,
    formState: { errors, isSubmitting },
  } = useForm<NewImageValues, unknown, NewImageOutput>({
    resolver: zodResolver(newImageSchema),
    defaultValues: {
      image: "",
      port: ENVIRONMENT_SETTINGS_DEFAULTS.port,
      regions: [],
      size: {
        cpuMillicores: ENVIRONMENT_SETTINGS_DEFAULTS.cpuMillicores,
        memoryMib: ENVIRONMENT_SETTINGS_DEFAULTS.memoryMib,
      },
    },
  });
  const regions = useWatch({ control, name: "regions" });
  const size = useWatch({ control, name: "size" });
  const image = useWatch({ control, name: "image" });

  useEffect(() => {
    if (availableRegions && getValues("regions").length === 0) {
      setValue(
        "regions",
        availableRegions.filter((r) => r.canSchedule).map((r) => r.name),
      );
    }
  }, [availableRegions, getValues, setValue]);

  const onValid = async (values: NewImageOutput) => {
    const created = await ensureApp("oci", () =>
      createImageApp({
        baseName: appNameFromImage(values.image),
        imageReference: values.image,
        settings: { regionNames: values.regions, port: values.port, ...values.size },
      }),
    );
    if (created.ok) {
      dispatch({ type: "next" });
    }
  };

  const formId = useId();
  return (
    <form id={formId} className="flex flex-1 flex-col gap-3" onSubmit={handleSubmit(onValid)}>
      <FieldStack>
        <Field title="Image">
          <ImageInput
            autoFocus
            registration={register("image")}
            value={image}
            error={errors.image?.message}
          />
        </Field>
        <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,2fr)] gap-4">
          <Field title={settingTitle.port}>
            <FormInput
              aria-label="Port"
              data-1p-ignore
              autoComplete="off"
              type="number"
              inputMode="numeric"
              error={errors.port?.message}
              {...register("port", { valueAsNumber: true })}
            />
          </Field>
          <Field title={settingTitle.regions}>
            <RegionSelect
              regions={regions}
              error={errors.regions?.message}
              onChange={(next) => setValue("regions", next, { shouldValidate: true })}
            />
          </Field>
        </div>
        <Field title={settingTitle.size}>
          <SizeField size={size} onChange={(next) => setValue("size", next)} />
        </Field>
      </FieldStack>
      <PaneActions>
        <Button
          type="submit"
          form={formId}
          variant="primary"
          size="sm"
          className="px-3"
          loading={isSubmitting}
          disabled={isSubmitting}
        >
          Continue
        </Button>
      </PaneActions>
    </form>
  );
}

const editImageSchema = z.object({ image: imageSchema });

function EditImage({
  projectId,
  appId,
  current,
  onDone,
}: { projectId: string; appId: string; current: string; onDone: () => void }) {
  const { updateImage } = useAppLifecycle(projectId);
  const {
    register,
    handleSubmit,
    control,
    formState: { errors, isSubmitting },
  } = useForm<z.input<typeof editImageSchema>, unknown, z.output<typeof editImageSchema>>({
    resolver: zodResolver(editImageSchema),
    defaultValues: { image: current },
  });
  const image = useWatch({ control, name: "image" });

  const onValid = async (values: z.output<typeof editImageSchema>) => {
    try {
      if (values.image !== current) {
        await updateImage(appId, values.image);
      }
      onDone();
    } catch (error) {
      toast.error("Could not change the image", { description: getErrorMessage(error) });
    }
  };

  return (
    <form
      className="flex flex-col gap-2 rounded-lg border border-grayA-4 p-3"
      onSubmit={(event) => {
        event.stopPropagation();
        return handleSubmit(onValid)(event);
      }}
    >
      <ImageInput
        autoFocus
        registration={register("image")}
        value={image}
        error={errors.image?.message}
      />
      <div className="flex justify-end gap-2">
        <Button type="button" size="sm" variant="ghost" onClick={onDone}>
          Cancel
        </Button>
        <Button type="submit" size="sm" variant="primary" loading={isSubmitting}>
          Save image
        </Button>
      </div>
    </form>
  );
}

function SavedImage({
  projectId,
  appId,
  onContinue,
}: { projectId: string; appId: string; onContinue: () => void }) {
  const [editing, setEditing] = useState(false);
  const { data } = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  const imageReference = data.at(0)?.imageReference ?? null;

  return (
    <div className="flex flex-1 flex-col gap-4">
      {editing && imageReference !== null ? (
        <EditImage
          projectId={projectId}
          appId={appId}
          current={imageReference}
          onDone={() => setEditing(false)}
        />
      ) : (
        <div className="flex items-center gap-3 rounded-lg border border-grayA-4 px-3 py-2.5">
          <IconCubeOutline18 className="size-4 shrink-0 text-gray-12" />
          <span className="min-w-0 flex-1 truncate font-mono text-xs text-gray-12">
            {imageReference ?? "Loading…"}
          </span>
          <Button
            size="sm"
            variant="ghost"
            disabled={imageReference === null}
            onClick={() => setEditing(true)}
          >
            Change image
          </Button>
        </div>
      )}
      <AppSettingsForm projectId={projectId} appId={appId} source="oci" onSaved={onContinue} />
    </div>
  );
}
