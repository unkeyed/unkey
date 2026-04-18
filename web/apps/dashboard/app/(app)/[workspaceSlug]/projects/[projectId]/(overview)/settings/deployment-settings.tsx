"use client";

import { CircleHalfDottedClock, Gear, Microchip } from "@unkey/icons";
import { SettingCardGroup } from "@unkey/ui";
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
import { SentinelSettings } from "./components/sentinel-settings";
import { SettingsGroup } from "./components/shared/settings-group";

// build is only required to invalidate other defaults. E.g onboarding settings, passes build=true to prevent expanding other sections.
type DeploymentSection = "advanced" | "sentinel" | "runtime" | "build";

type DeploymentSettingsProps = {
  githubReadOnly?: boolean;
  sections?: Partial<Record<DeploymentSection, true>>;
  onBeforeNavigate?: () => void;
};

export const DeploymentSettings = ({
  githubReadOnly = false,
  sections = { build: true, runtime: true, advanced: true, sentinel: true },
  onBeforeNavigate,
}: DeploymentSettingsProps) => {
  return (
    <div className="flex flex-col gap-6">
      <SettingCardGroup>
        <GitHub readOnly={githubReadOnly} onBeforeNavigate={onBeforeNavigate} />
        <RootDirectory />
        <Dockerfile />
        <WatchPaths />
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
        icon={<Microchip iconSize="md-medium" />}
        title="Sentinels"
        defaultExpanded={Boolean(sections.sentinel)}
      >
        <SentinelSettings />
      </SettingsGroup>
      <SettingsGroup
        icon={<Gear iconSize="md-medium" />}
        title="Advanced configurations"
        defaultExpanded={Boolean(sections.advanced)}
      >
        <SettingCardGroup>
          <CustomDomains />
          <OpenapiSpecPath />
          <UpstreamProtocol />
        </SettingCardGroup>
      </SettingsGroup>
    </div>
  );
};
