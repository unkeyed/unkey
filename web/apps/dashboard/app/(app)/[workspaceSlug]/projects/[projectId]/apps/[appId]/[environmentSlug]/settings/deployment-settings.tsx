"use client";

import { collection } from "@/lib/collections";
import { ociImageReferenceSchema } from "@/lib/collections/deploy/apps";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { zodResolver } from "@hookform/resolvers/zod";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { useMutation } from "@tanstack/react-query";
import { IconCodeBranchOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  FormInput,
  SettingCardGroup,
  SettingsGroup,
  toast,
} from "@unkey/ui";
import Link from "next/link";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useAppId, useProjectData } from "../../data-provider";
import { useAppScope } from "../environment-context";
import { OpenapiSpecPath } from "./components/advanced-settings/openapi-spec-path";
import { UpstreamProtocol } from "./components/advanced-settings/upstream-protocol";
import { AutoDeploy } from "./components/build-settings/auto-deploy-settings";
import { BuildCommand } from "./components/build-settings/build-command-settings";
import { Dockerfile } from "./components/build-settings/dockerfile-settings";
import { GitHub } from "./components/build-settings/github-settings";
import { RootDirectory } from "./components/build-settings/root-directory-settings";
import { WatchPaths } from "./components/build-settings/watch-paths-settings";
import { Command } from "./components/runtime-settings/command";
import { Cpu } from "./components/runtime-settings/cpu";
import { Healthcheck } from "./components/runtime-settings/healthcheck";
import { Instances } from "./components/runtime-settings/instances";
import { Memory } from "./components/runtime-settings/memory";
import { Port } from "./components/runtime-settings/port-settings";
import { Regions } from "./components/runtime-settings/regions";
import { Storage } from "./components/runtime-settings/storage";
import { SettingField } from "./components/shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "./components/shared/form-setting-card";
import { SettingsSection } from "./components/shared/settings-section";

const NEXT_DEPLOY = "Changes apply on next deploy";

function useBuildSource() {
  const { projectId } = useProjectData();
  const appId = useAppId();
  const appQuery = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  const app = appQuery.data?.[0];
  const shouldLoadGitHub = app
    ? match(app.sourceType)
        .with("git", () => true)
        .with("oci", () => false)
        .with("unknown", () => true)
        .exhaustive()
    : false;
  const { data } = trpc.github.getInstallations.useQuery(
    { projectId, appId },
    { enabled: shouldLoadGitHub },
  );

  const hasRepository = app
    ? match(app.sourceType)
        .with("oci", () => false)
        .with("git", () => !data || Boolean(data.repoConnection?.repositoryFullName))
        .with("unknown", () => Boolean(data?.repoConnection?.repositoryFullName))
        .exhaustive()
    : false;

  return { projectId, appId, app, hasRepository };
}

export function ComputeSettings() {
  return (
    <SettingsGroup title="Resources" pendingNote={NEXT_DEPLOY}>
      <Regions />
      <Instances />
      <Cpu />
      <Memory />
      <Storage />
    </SettingsGroup>
  );
}

export function DeploySettings() {
  const { app, hasRepository } = useBuildSource();
  const scope = useAppScope();
  if (!app) {
    return null;
  }
  if (!hasRepository) {
    return (
      <SettingsSection>
        <SettingCardGroup>
          <EmptyState frame="none">
            <EmptyStateIcon>
              <IconCodeBranchOutline18 />
            </EmptyStateIcon>
            <EmptyStateHeader>
              <EmptyStateTitle>No repository connected</EmptyStateTitle>
              <EmptyStateDescription>Connect a repository to deploy on push.</EmptyStateDescription>
            </EmptyStateHeader>
            <EmptyStateActions>
              <Button
                variant="outline"
                render={<Link href={routes.projects.apps.settings({ ...scope, page: "build" })} />}
              >
                Go to Build
              </Button>
            </EmptyStateActions>
          </EmptyState>
        </SettingCardGroup>
      </SettingsSection>
    );
  }
  return (
    <SettingsSection>
      <SettingCardGroup>
        <AutoDeploy />
      </SettingCardGroup>
    </SettingsSection>
  );
}

export function BuildSettings({ githubReadOnly = false }: { githubReadOnly?: boolean }) {
  const { projectId, appId, app, hasRepository } = useBuildSource();

  return (
    <SettingsGroup pendingNote={NEXT_DEPLOY}>
      {app
        ? match(app.sourceType)
            .with("oci", () => (
              <OCIImage
                projectId={projectId}
                appId={appId}
                imageReference={app.imageReference ?? ""}
              />
            ))
            .with("git", () => <GitHub readOnly={githubReadOnly} />)
            .with("unknown", () => <GitHub readOnly={githubReadOnly} />)
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
    </SettingsGroup>
  );
}

export function RuntimeSettings() {
  return (
    <>
      <SettingsGroup title="Process" pendingNote={NEXT_DEPLOY}>
        <Port />
        <Command />
      </SettingsGroup>
      <SettingsGroup title="Health check" pendingNote={NEXT_DEPLOY}>
        <Healthcheck />
      </SettingsGroup>
    </>
  );
}

export function AdvancedSettings() {
  return (
    <>
      <SettingsGroup title="API" pendingNote={NEXT_DEPLOY}>
        <OpenapiSpecPath />
      </SettingsGroup>
      <SettingsGroup title="Networking" pendingNote={NEXT_DEPLOY}>
        <UpstreamProtocol />
      </SettingsGroup>
    </>
  );
}

const ociImageFormSchema = z.object({
  imageReference: ociImageReferenceSchema,
});

const OCIImage = ({
  projectId,
  appId,
  imageReference,
}: { projectId: string; appId: string; imageReference: string }) => {
  const updateImage = useMutation({
    mutationFn: (image: string) =>
      getUnkeyClient().apps.updateApp({
        project: projectId,
        app: appId,
        oci: { image },
      }),
  });
  const {
    register,
    handleSubmit,
    reset,
    control,
    formState: { errors, isSubmitting, isValid },
  } = useForm<z.infer<typeof ociImageFormSchema>>({
    resolver: zodResolver(ociImageFormSchema),
    mode: "onChange",
    defaultValues: { imageReference },
  });

  useEffect(() => {
    reset({ imageReference });
  }, [imageReference, reset]);

  const currentImageReference = useWatch({ control, name: "imageReference" });
  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [
      currentImageReference === imageReference,
      { status: "disabled", reason: "No changes to save" },
    ],
  ]);

  const onSubmit = async (values: z.infer<typeof ociImageFormSchema>) => {
    try {
      await updateImage.mutateAsync(values.imageReference);
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
      description="Image used for new deployments."
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
    >
      <SettingField>
        <FormInput
          aria-label="Image reference"
          placeholder="ghcr.io/acme/app:v1.2.3"
          error={errors.imageReference?.message}
          {...register("imageReference")}
        />
      </SettingField>
    </FormSettingCard>
  );
};
