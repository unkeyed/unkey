"use client";

import { CircleHalfDottedClock, Gear, StackPerspective2 } from "@unkey/icons";
import { SettingCardGroup } from "@unkey/ui";

import { Dockerfile } from "./components/build-settings/dockerfile-settings";
import { GitHub } from "./components/build-settings/github-settings";
import { RootDirectory } from "./components/build-settings/root-directory-settings";

import { Command } from "./components/runtime-settings/command";
import { Cpu } from "./components/runtime-settings/cpu";
import { Healthcheck } from "./components/runtime-settings/healthcheck";
import { Instances } from "./components/runtime-settings/instances";
import { Memory } from "./components/runtime-settings/memory";
import { Port } from "./components/runtime-settings/port-settings";
import { Regions } from "./components/runtime-settings/regions";

import { CustomDomains } from "./components/advanced-settings/custom-domains";
import { EnvVars } from "./components/advanced-settings/env-vars";

import { Keyspaces } from "./components/sentinel-settings/keyspaces";
import { SettingsGroup } from "./components/shared/settings-group";

// build is only required to invalidate other defaults. E.g onboarding settings, passes build=true to prevent expanding other sections.
type DeploymentSection = "advanced" | "sentinel" | "runtime" | "build";

type DeploymentSettingsProps = {
  githubReadOnly?: boolean;
  sections?: Partial<Record<DeploymentSection, true>>;
};


export const DeploymentSettings = ({ githubReadOnly = false, sections = { build: true, runtime: true, advanced: true, sentinel: true } }: DeploymentSettingsProps) => {
  return (
    <div className="flex flex-col gap-6">
      <SettingCardGroup>
        <GitHub readOnly={githubReadOnly} />
        <RootDirectory />
        <Dockerfile />
      </SettingCardGroup>
      <SettingsGroup icon={<CircleHalfDottedClock iconSize="md-medium" />} title="Runtime settings" defaultExpanded={Boolean(sections.runtime)}>
        <SettingCardGroup>
          <Regions />
          <Instances />
          <Cpu />
          <Memory />
          {/* Temporarily disabled */}
          {/* <Storage /> */}
          <Healthcheck />
          <Port />
          <Command />
          {/* Temporarily disabled */}
          {/* <Scaling /> */}
        </SettingCardGroup>
      </SettingsGroup>
      <SettingsGroup icon={<Gear iconSize="md-medium" />} title="Advanced configurations" defaultExpanded={Boolean(sections.advanced)}>
        <SettingCardGroup>
          <EnvVars />
          <CustomDomains />
        </SettingCardGroup>
      </SettingsGroup>
      <SettingsGroup icon={<StackPerspective2 iconSize="md-medium" />} title="Sentinel configurations" defaultExpanded={Boolean(sections.sentinel)}>
        <SettingCardGroup>
          <Keyspaces />
        </SettingCardGroup>
      </SettingsGroup>
    </div>
  )
}
