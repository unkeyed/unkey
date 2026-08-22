"use client";

import { collection } from "@/lib/collections";
import { dockerImageReferenceSchema } from "@/lib/collections/deploy/apps";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { CircleHalfDottedClock, Docker, Gear } from "@unkey/icons";
import { match } from "@unkey/match";
import { FormInput, SettingCardGroup, toast } from "@unkey/ui";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useAppId, useProjectData } from "../data-provider";
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

import { CustomDomains } from "./components/advanced-settings/custom-domains";
import { OpenapiSpecPath } from "./components/advanced-settings/openapi-spec-path";
import { UpstreamProtocol } from "./components/advanced-settings/upstream-protocol";
import { SettingField } from "./components/shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "./components/shared/form-setting-card";
import { SettingsGroup } from "./components/shared/settings-group";

// build is only required to invalidate other defaults. E.g onboarding settings, passes build=true to prevent expanding other sections.
type DeploymentSection = "advanced" | "runtime" | "build";

type DeploymentSettingsProps = {
  githubReadOnly?: boolean;
  sections?: Partial<Record<DeploymentSection, true>>;
  onBeforeNavigate?: () => void;
};

export const DeploymentSettings = ({
  githubReadOnly = false,
  sections = { build: true, runtime: true, advanced: true },
  onBeforeNavigate,
}: DeploymentSettingsProps) => {
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
        .with("docker", () => false)
        // Legacy apps retain the old connect/disconnect flow until every
        // caller creates an app with an explicit source.
        .with("unknown", () => true)
        .exhaustive()
    : false;
  const { data } = trpc.github.getInstallations.useQuery(
    { projectId, appId },
    { enabled: shouldLoadGitHub },
  );

  const sourceSettings = app
    ? match(app.sourceType)
        .with("docker", () => (
          <DockerImage appId={appId} imageReference={app.imageReference ?? ""} />
        ))
        .with("git", () => <GitHub readOnly={githubReadOnly} onBeforeNavigate={onBeforeNavigate} />)
        .with("unknown", () => (
          <GitHub readOnly={githubReadOnly} onBeforeNavigate={onBeforeNavigate} />
        ))
        .exhaustive()
    : null;
  const showBuildSettings = app
    ? match(app.sourceType)
        .with("docker", () => false)
        .with("git", () => !data || Boolean(data.repoConnection?.repositoryFullName))
        .with("unknown", () => Boolean(data?.repoConnection?.repositoryFullName))
        .exhaustive()
    : false;

  return (
    <div className="flex flex-col gap-6">
      <SettingCardGroup>
        {sourceSettings}
        {showBuildSettings ? (
          <>
            <RootDirectory />
            <Dockerfile />
            <BuildCommand />
            <WatchPaths />
            <AutoDeploy />
          </>
        ) : null}
      </SettingCardGroup>
      <SettingsGroup
        icon={<CircleHalfDottedClock iconSize="md-medium" />}
        title="Runtime settings"
        defaultExpanded={Boolean(sections.runtime)}
      >
        <SettingCardGroup>
          <Regions />
          <Instances />
          <Cpu />
          <Memory />
          <Storage />
          <Healthcheck />
          <Port />
          <Command />
          {/* Temporarily disabled */}
          {/* <Scaling /> */}
        </SettingCardGroup>
      </SettingsGroup>
      <SettingsGroup
        icon={<Gear iconSize="md-medium" />}
        title="Advanced configurations"
        defaultExpanded={Boolean(sections.advanced)}
      >
        <SettingCardGroup>
          <div id="custom-domains" className="scroll-mt-24">
            <CustomDomains />
          </div>
          <OpenapiSpecPath />
          <UpstreamProtocol />
        </SettingCardGroup>
      </SettingsGroup>
    </div>
  );
};

const dockerImageFormSchema = z.object({
  imageReference: dockerImageReferenceSchema,
});

const DockerImage = ({ appId, imageReference }: { appId: string; imageReference: string }) => {
  const updateImage = trpc.deploy.app.updateDockerSource.useMutation();
  const {
    register,
    handleSubmit,
    reset,
    control,
    formState: { errors, isSubmitting, isValid },
  } = useForm<z.infer<typeof dockerImageFormSchema>>({
    resolver: zodResolver(dockerImageFormSchema),
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

  const onSubmit = async (values: z.infer<typeof dockerImageFormSchema>) => {
    try {
      await updateImage.mutateAsync({ appId, imageReference: values.imageReference });
      reset(values);
      await collection.apps.utils.refetch();
      toast.success("Docker image updated");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to update Docker image");
    }
  };

  return (
    <FormSettingCard
      icon={<Docker className="text-gray-12" iconSize="xl-regular" />}
      title="Image"
      description="Default image reference for new deployments"
      displayValue={<span className="font-mono text-xs">{imageReference}</span>}
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
    >
      <SettingField>
        <FormInput
          label="Image reference"
          requirement="required"
          description="Include a tag or digest. Saving changes the default for future deployments; it does not replace the running deployment."
          placeholder="ghcr.io/acme/app:v1.2.3"
          error={errors.imageReference?.message}
          {...register("imageReference")}
        />
      </SettingField>
    </FormSettingCard>
  );
};
