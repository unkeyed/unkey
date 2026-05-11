"use client";

import { Grid } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import Link from "next/link";

type ExtensionsHeaderProps = {
  basePath: string;
  active: "marketplace" | "installed" | "submit";
  installedCount: number;
};

export function ExtensionsHeader({ basePath, active, installedCount }: ExtensionsHeaderProps) {
  return (
    <div className="flex flex-col gap-5">
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3">
          <div className="size-10 bg-gray-3 rounded-[10px] flex items-center justify-center shrink-0 shadow-sm shadow-grayA-8/20 dark:ring-1 dark:ring-gray-4 dark:shadow-none">
            <Grid iconSize="xl-medium" className="size-5 text-accent-12" />
          </div>
          <div className="flex flex-col gap-0.5">
            <h1 className="font-semibold text-gray-12 text-lg leading-8">Extensions</h1>
            <p className="text-[13px] text-gray-11 leading-5">
              Wire your project into the rest of your stack — logging, alerting, analytics, and
              more.
            </p>
          </div>
        </div>

        <SegmentedTabs basePath={basePath} active={active} installedCount={installedCount} />
      </div>
    </div>
  );
}

function SegmentedTabs({
  basePath,
  active,
  installedCount,
}: {
  basePath: string;
  active: ExtensionsHeaderProps["active"];
  installedCount: number;
}) {
  return (
    <nav className="flex items-center gap-0.5 rounded-lg border border-grayA-4 bg-grayA-2 p-0.5 shrink-0">
      <SegmentLink href={basePath} active={active === "marketplace"}>
        Marketplace
      </SegmentLink>
      <SegmentLink href={`${basePath}/installed`} active={active === "installed"}>
        Installed
        <span
          className={cn(
            "ml-1.5 rounded-full px-1.5 text-[10px] tabular-nums leading-4",
            active === "installed" ? "bg-grayA-3 text-gray-12" : "bg-grayA-3 text-gray-11",
          )}
        >
          {installedCount}
        </span>
      </SegmentLink>
      <SegmentLink href={`${basePath}/submit`} active={active === "submit"}>
        Submit
      </SegmentLink>
    </nav>
  );
}

function SegmentLink({
  href,
  active,
  children,
}: {
  href: string;
  active: boolean;
  children: React.ReactNode;
}) {
  return (
    <Link
      href={href}
      className={cn(
        "inline-flex items-center rounded-md px-2.5 py-1 text-[12px] font-medium transition-colors",
        active
          ? "bg-background text-accent-12 shadow-sm shadow-grayA-8/10 dark:ring-1 dark:ring-gray-4"
          : "text-gray-11 hover:text-gray-12",
      )}
    >
      {children}
    </Link>
  );
}
