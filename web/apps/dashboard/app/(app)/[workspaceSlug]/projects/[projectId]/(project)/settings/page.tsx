"use client";

import { SettingsDangerZone, SettingsShell } from "@unkey/ui";
import { DeleteProject } from "./components/delete-project";
import { UpdateProjectSettings } from "./components/update-project-settings";

export default function ProjectSettingsPage() {
  return (
    <>
      <SettingsShell>
        <div className="flex flex-col gap-2 items-center">
          <span className="font-semibold text-gray-12 leading-8 text-lg">Project settings</span>
          <span className="leading-4 text-gray-11 text-[13px]">
            Manage your project name.
          </span>
        </div>
        <div className="w-full">
          <UpdateProjectSettings />
        </div>
      </SettingsShell>
      <div className="w-225 mx-auto mt-10 mb-14">
        <SettingsDangerZone>
          <DeleteProject />
        </SettingsDangerZone>
      </div>
    </>
  );
}
