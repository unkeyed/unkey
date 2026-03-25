"use client";
import { Shell } from "./shell";

export const SettingsClientSkeleton = () => {
  return (
    <Shell>
      <div className="flex flex-col gap-2 items-center">
        <span className="font-semibold text-gray-12 leading-8 text-lg">API Settings</span>
        <span className="leading-4 text-gray-11 text-[13px]">
          Configure your API name, default key settings, and access controls.
        </span>
      </div>
      <div className="w-full h-[180px] bg-grayA-3 rounded-lg animate-pulse" />
      <div className="w-full h-[130px] bg-grayA-3 rounded-lg animate-pulse" />
      <div className="w-full h-[90px] bg-grayA-3 rounded-lg animate-pulse" />
      <div className="w-full h-[120px] bg-grayA-3 rounded-lg animate-pulse" />
    </Shell>
  );
};
