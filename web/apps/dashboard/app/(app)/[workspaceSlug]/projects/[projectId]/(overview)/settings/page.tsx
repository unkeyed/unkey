"use client";

import { CircleHalfDottedClock } from "@unkey/icons";
import { DockerfileSettings } from "./components/basic-settings/dockerfile-settings";
import { PortSettings } from "./components/basic-settings/port-settings";
import { GitHubSettings } from "./components/basic-settings/github-settings";
import { RootDirectorySettings } from "./components/basic-settings/root-directory-settings";
import { Cpu } from "./components/runtime-settings/cpu";
import { Instances } from "./components/runtime-settings/instances";
import { Scaling } from "./components/runtime-settings/scaling";
import { Memory } from "./components/runtime-settings/memory";
import { Regions } from "./components/runtime-settings/regions";
import { Storage } from "./components/runtime-settings/storage";
import { Healthcheck } from "./components/runtime-settings/healthcheck";
import { SettingCardGroup } from "@unkey/ui";
import { SettingsGroup } from "./components/shared/settings-group";

export default function SettingsPage() {
  return (
    <div className="w-[900px] flex flex-col justify-center items-center gap-6 mx-auto mt-14">
      <div className="flex flex-col gap-2 items-center">
        <span className="font-semibold text-gray-12 leading-8 text-lg">Configure deployment</span>
        <span className="leading-4 text-gray-11 text-[13px]">
          Review the defaults. Edit anything you'd like to adjust.
        </span>
      </div>
      <div className="flex flex-col gap-6">
        <div className="flex flex-col w-full">
          <SettingCardGroup>
            <GitHubSettings />
            <RootDirectorySettings />
            <DockerfileSettings />
            <PortSettings />
          </SettingCardGroup>
        </div>
        <SettingsGroup
          icon={<CircleHalfDottedClock iconSize="md-medium" />}
          title="Runtime settings"
        >
          <SettingCardGroup>
            <Regions />
            <Instances />
            <Cpu />
            <Memory />
            {/* Temporarily disabled */}
            {/* <Storage /> */}
            <Healthcheck />
            {/* Temporarily disabled */}
            {/* <Scaling /> */}
          </SettingCardGroup>
        </SettingsGroup>
      </div>
    </div>
  );
}
