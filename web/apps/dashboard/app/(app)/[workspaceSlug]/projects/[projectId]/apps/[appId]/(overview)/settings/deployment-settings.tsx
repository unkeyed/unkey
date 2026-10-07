"use client";

import { collection } from "@/lib/collections";
import { ociImageReferenceSchema } from "@/lib/collections/deploy/apps";
import { NEXT_DEPLOY } from "@/lib/collections/deploy/pending-redeploy";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { zodResolver } from "@hookform/resolvers/zod";
import { match } from "@unkey/match";
import {
  FormInput,
  SettingsGroup,
  SettingsGroupContent,
  SettingsGroupTitle,
  formSaveState,
  toast,
} from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { OpenapiSpecPath } from "./components/advanced-settings/openapi-spec-path";
import { UpstreamProtocol } from "./components/advanced-settings/upstream-protocol";
import { AppName } from "./components/app-name";
import { BuildCommand } from "./components/build-settings/build-command-settings";
import { Dockerfile } from "./components/build-settings/dockerfile-settings";
import { GitHub } from "./components/build-settings/github-settings";
import { RootDirectory } from "./components/build-settings/root-directory-settings";
import { WatchPaths } from "./components/build-settings/watch-paths-settings";
import { Command } from "./components/runtime-settings/command";
import { Healthcheck } from "./components/runtime-settings/healthcheck";
import { Port } from "./components/runtime-settings/port-settings";
import { SettingField } from "./components/shared/form-blocks";
import { FormSettingCard } from "./components/shared/form-setting-card";
import { useApp, useBuildSource } from "./hooks/use-build-source";

export function GeneralSettings() {
  const { projectId, appId, app } = useApp();
  if (!app) {
    return null;
  }
  return (
    <SettingsGroup>
      <SettingsGroupContent>
        <AppName projectId={projectId} appId={appId} name={app.name} />
      </SettingsGroupContent>
    </SettingsGroup>
  );
}

export function BuildSettings({ githubReadOnly = false }: { githubReadOnly?: boolean }) {
  const { projectId, appId, app, hasRepository } = useBuildSource();

  return (
    <SettingsGroup>
      <SettingsGroupContent pendingNote={NEXT_DEPLOY}>
        {app
          ? match(app.sourceType)
              .with("oci", () => (
                <OCIImage
                  projectId={projectId}
                  appId={appId}
                  imageReference={app.imageReference ?? ""}
                />
              ))
              .with("git", "unknown", () => <GitHub readOnly={githubReadOnly} />)
              .exhaustive()
          : null}
        {hasRepository ? (
          <>
            <RootDirectory />
            <Dockerfile />
            <BuildCommand />
            <WatchPaths />
          </>
        ) : null}
      </SettingsGroupContent>
    </SettingsGroup>
  );
}

export function RuntimeSettings() {
  return (
    <>
      <SettingsGroup>
        <SettingsGroupTitle>Process</SettingsGroupTitle>
        <SettingsGroupContent pendingNote={NEXT_DEPLOY}>
          <Port />
          <Command />
        </SettingsGroupContent>
      </SettingsGroup>
      <SettingsGroup>
        <SettingsGroupTitle>Health check</SettingsGroupTitle>
        <SettingsGroupContent pendingNote={NEXT_DEPLOY}>
          <Healthcheck />
        </SettingsGroupContent>
      </SettingsGroup>
    </>
  );
}

export function AdvancedSettings() {
  return (
    <>
      <SettingsGroup>
        <SettingsGroupTitle>API</SettingsGroupTitle>
        <SettingsGroupContent pendingNote={NEXT_DEPLOY}>
          <OpenapiSpecPath />
        </SettingsGroupContent>
      </SettingsGroup>
      <SettingsGroup>
        <SettingsGroupTitle>Networking</SettingsGroupTitle>
        <SettingsGroupContent pendingNote={NEXT_DEPLOY}>
          <UpstreamProtocol />
        </SettingsGroupContent>
      </SettingsGroup>
    </>
  );
}

const ociImageFormSchema = z.object({
  imageReference: ociImageReferenceSchema,
});

function OCIImage({
  projectId,
  appId,
  imageReference,
}: { projectId: string; appId: string; imageReference: string }) {
  const {
    register,
    handleSubmit,
    reset,
    control,
    formState: { errors, isSubmitting, isValid },
  } = useForm<z.infer<typeof ociImageFormSchema>>({
    resolver: zodResolver(ociImageFormSchema),
    mode: "onChange",
    values: { imageReference },
  });

  const currentImageReference = useWatch({ control, name: "imageReference" });
  const dirty = currentImageReference !== imageReference;
  const saveState = formSaveState({
    isSubmitting,
    isValid,
    isDirty: dirty,
  });

  const onSubmit = async (values: z.infer<typeof ociImageFormSchema>) => {
    try {
      await getUnkeyClient().apps.updateApp({
        project: projectId,
        app: appId,
        oci: { image: values.imageReference },
      });
      reset(values);
      await collection.apps.utils.refetch();
      toast.success("Container image updated");
    } catch (error) {
      toast.error(getErrorMessage(error, "Failed to update container image"));
    }
  };

  return (
    <FormSettingCard
      title="Image"
      onSubmit={handleSubmit(onSubmit)}
      dirty={dirty}
      saveState={saveState}
    >
      <SettingField>
        <FormInput
          data-1p-ignore
          aria-label="Image reference"
          placeholder="ghcr.io/acme/app:v1.2.3"
          error={errors.imageReference?.message}
          {...register("imageReference")}
        />
      </SettingField>
    </FormSettingCard>
  );
}
