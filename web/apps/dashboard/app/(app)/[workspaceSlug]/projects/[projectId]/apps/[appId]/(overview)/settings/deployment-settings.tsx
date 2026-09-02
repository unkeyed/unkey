"use client";

import { SettingCardGroup } from "@unkey/ui";
import { IconCircleHalfDottedClockOutline18, IconGearOutline18 } from "nucleo-ui-outline-18";
import { trpc } from "@/lib/trpc/client";
import { useAppId, useProjectData } from "../data-provider";
import { CustomDomains } from "./components/advanced-settings/custom-domains";
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
  const { data } = trpc.github.getInstallations.useQuery({ projectId, appId });

  // An app's source is fixed at creation: a repo connection means git, its
  // absence means a published image. Nothing here can switch between them, so
  // the whole group is hidden for image apps rather than offering a repo.
  const isGitSourced = !data || Boolean(data.repoConnection?.repositoryFullName);

  return (
    <div className="flex flex-col gap-6">
      {isGitSourced ? (
        <SettingCardGroup>
          <GitHub readOnly={githubReadOnly} onBeforeNavigate={onBeforeNavigate} />
          <RootDirectory />
          <Dockerfile />
          <BuildCommand />
          <WatchPaths />
          <AutoDeploy />
        </SettingCardGroup>
      ) : null}
      <SettingsGroup
        icon={<IconCircleHalfDottedClockOutline18 className="size-4" />}
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
        icon={<IconGearOutline18 className="size-4" />}
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
