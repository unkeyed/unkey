"use client";

import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/data-provider";
import { Button, Skeleton } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";

const ROWS = [
  { titleW: "w-16", descW: "w-52", controlW: "w-44" },
  { titleW: "w-24", descW: "w-80", controlW: "w-64" },
  { titleW: "w-16", descW: "w-72", controlW: "w-52" },
];

export const ConfigureDeploymentFallback = ({ settingsReady }: { settingsReady: boolean }) => {
  const { isEnvironmentsLoading } = useProjectData();
  if (!isEnvironmentsLoading && settingsReady) {
    return null;
  }

  return (
    <div className="w-225">
      <div className="flex flex-col gap-3">
        <Skeleton className="mx-1 h-4 w-28 rounded" />
        <div className="border bg-raised rounded-lg overflow-hidden divide-y divide-grayA-4">
          {ROWS.map(({ titleW, descW, controlW }) => (
            <div key={titleW + descW} className="flex flex-col gap-4 px-5 py-5 lg:flex-row">
              <div className="flex shrink-0 flex-col gap-2 lg:w-2/5">
                <Skeleton className={cn("h-4 rounded", titleW)} />
                <Skeleton className={cn("h-3 rounded", descW)} />
              </div>
              <Skeleton className={cn("h-8 rounded-md", controlW)} />
            </div>
          ))}
          <div className="flex justify-end bg-grayA-2 px-5 py-3">
            <Skeleton className="h-7 w-28 rounded-md" />
          </div>
        </div>
      </div>

      <div className="flex justify-end mt-6 mb-10 flex-col gap-4">
        <Button type="button" variant="primary" size="xlg" className="rounded-lg" disabled>
          Deploy
        </Button>
        <span className="text-gray-10 text-sm text-center">
          We'll build your image, provision infrastructure, and more.
          <br />
        </span>
      </div>
    </div>
  );
};
