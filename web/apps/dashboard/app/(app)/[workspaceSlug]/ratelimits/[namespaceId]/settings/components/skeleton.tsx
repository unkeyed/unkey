"use client";

import { Skeleton } from "@unkey/ui";

export const SettingsClientSkeleton = () => {
  return (
    <>
      <Skeleton className="w-full h-[150px] rounded-lg" />
      <Skeleton className="w-full h-[120px] rounded-lg" />
    </>
  );
};
