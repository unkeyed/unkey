"use client";

import { FileSettings, FolderLink } from "@unkey/icons";
import { SettingCard } from "@unkey/ui";
import { GitHubSettings } from "./components/github-settings";

export default function SettingsPage() {
  return (
    <div className="w-[900px] flex flex-col justify-center items-center gap-5 mx-auto mt-14">
      <div className="flex flex-col gap-2 items-center">
        <span className="font-semibold text-gray-12 leading-8 text-lg">Configure deployment</span>
        <span className="leading-4 text-gray-11 text-[13px]">
          Review the defaults. Edit anything you'd like to adjust.
        </span>
      </div>
      <div className="flex flex-col w-full">
        <GitHubSettings />
        <SettingCard
          className="px-4 py-[18px]"
          icon={<FileSettings className="text-gray-12" iconSize="xl-regular" />}
          title="Dockerfile"
          description="Dockerfile location used for docker build. (e.g., services/api/Dockerfile)"
          contentWidth="w-full lg:w-[320px] justify-end"
        >
          <div>asdsd</div>
        </SettingCard>
        <SettingCard
          className="px-4 py-[18px]"
          icon={<FolderLink className="text-gray-12" iconSize="xl-regular" />}
          title="Root directory"
          description="Build context directory. All COPY/ADD commands are relative to this path. (e.g., services/api)"
          border="bottom"
          contentWidth="w-full lg:w-[320px] justify-end"
        >
          <div>asdsd</div>
        </SettingCard>
      </div>
    </div>
  );
}
