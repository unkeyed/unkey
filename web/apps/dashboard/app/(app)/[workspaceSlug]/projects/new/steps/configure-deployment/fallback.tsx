"use client";

import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useProjectData } from "../../../[projectId]/(overview)/data-provider";

export const ConfigureDeploymentFallback = () => {
  const { isEnvironmentsLoading } = useProjectData();
  if (!isEnvironmentsLoading) {
    return null;
  }

  const cards = [
    { titleW: "w-16", descW: "w-52", badgeW: "w-36" },
    { titleW: "w-24", descW: "w-80", badgeW: "w-7" },
    { titleW: "w-16", descW: "w-72", badgeW: "w-20" },
  ];

  const sections = [{ titleW: "w-28" }, { titleW: "w-40" }, { titleW: "w-36" }];

  return (
    <div className="w-225">
      <div className="flex flex-col gap-6">
        {/* SettingCardGroup skeleton */}
        <div className="border border-grayA-4 rounded-[14px] overflow-hidden divide-y divide-grayA-4">
          {cards.map(({ titleW, descW, badgeW }, i) => (
            <div
              // biome-ignore lint/suspicious/noArrayIndexKey: safe to leave
              key={i}
              className="px-4 py-[18px] lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row"
            >
              <div className="flex gap-4 items-center">
                <div className="bg-grayA-3 size-8 rounded-[10px] shrink-0 animate-pulse dark:ring-1 dark:ring-gray-4 shadow-sm shadow-grayA-8/20" />
                <div className="flex flex-col gap-1 text-sm w-fit">
                  <div className={cn("h-[13px] rounded bg-grayA-3 animate-pulse", titleW)} />
                  <div className={cn("h-3 rounded bg-grayA-3 animate-pulse mt-0.5", descW)} />
                </div>
              </div>
              <div className="flex w-full lg:w-[320px] items-center gap-4 justify-end">
                <div
                  className={cn(
                    "h-7 rounded-md border border-grayA-4 bg-grayA-3 animate-pulse",
                    badgeW,
                  )}
                />
                <div className="size-3.5 rounded bg-grayA-3 animate-pulse" />
              </div>
            </div>
          ))}
        </div>

        {/* SettingsGroup collapsed headers skeleton */}
        {sections.map(({ titleW }, i) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: safe to leave
          <div key={i} className="flex flex-col">
            <div className="flex items-center justify-between mb-4 px-2">
              <div className="flex items-center gap-2.5">
                <div className="size-4 rounded bg-grayA-3 animate-pulse" />
                <div className={cn("h-3.5 rounded bg-grayA-3 animate-pulse", titleW)} />
              </div>
              <div className="flex items-center gap-1">
                <div className="h-3 w-8 rounded bg-grayA-3 animate-pulse" />
                <div className="size-3 rounded bg-grayA-3 animate-pulse" />
              </div>
            </div>
          </div>
        ))}
      </div>

      <div className="flex justify-end mt-6 mb-10 flex-col gap-4">
        <Button type="button" variant="primary" size="xlg" className="rounded-lg" disabled>
          Deploy
        </Button>
        <span className="text-gray-10 text-[13px] text-center">
          We'll build your image, provision infrastructure, and more.
          <br />
        </span>
      </div>
    </div>
  );
};
