"use client";
import { SettingsShell } from "@unkey/ui";

export const SettingsClientSkeleton = () => {
  return (
    <SettingsShell>
      <div className="flex flex-col gap-2 items-center">
        <span className="font-semibold text-gray-12 leading-8 text-lg">Ratelimit Settings</span>
        <span className="leading-4 text-gray-11 text-[13px]">
          Configure your ratelimit namespace name and settings.
        </span>
      </div>
      <div className="w-full h-[150px] bg-grayA-3 rounded-lg animate-pulse" />
      <div className="w-full h-[120px] bg-grayA-3 rounded-lg animate-pulse" />
    </SettingsShell>
  );
};
