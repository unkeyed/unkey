"use client";

import { Skeleton } from "@unkey/ui";

export const SettingsClientSkeleton = () => {
  return (
    <>
      <Skeleton className="w-full h-[180px] rounded-lg" />
      <Skeleton className="w-full h-[130px] rounded-lg" />
      <Skeleton className="w-full h-[90px] rounded-lg" />
      <Skeleton className="w-full h-[120px] rounded-lg" />
    </>
  );
};
