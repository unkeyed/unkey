import type React from "react";

export const SettingField = ({ children }: { children: React.ReactNode }) => (
  <div className="flex flex-col gap-1.5 max-w-(--setting-w) [&_label]:text-gray-11 [&_label]:text-[13px]">
    {children}
  </div>
);
