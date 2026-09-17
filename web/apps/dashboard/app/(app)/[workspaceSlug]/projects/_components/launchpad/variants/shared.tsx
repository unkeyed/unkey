"use client";

import { cn } from "@/lib/utils";
import { IconBoltOutline18, IconCubeOutline18, IconKey2Outline18 } from "@unkey/icons";
import { routes } from "@/lib/navigation/routes";
import { TimestampInfo } from "@unkey/ui";
import Link from "next/link";
import { useDeployState } from "../use-deploy-state";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { type Action, ActionRow, ProjectChip, Spark, fmt } from "../parts";
import type { LaunchpadRow, VariantProps } from "../types";

/**
 * A workspace with nothing to jump to gets things to start instead. Gateway
 * policies configure ratelimiting inline, so a namespace is never offered here.
 */
export function useGetStartedActions(): Action[] {
  const workspace = useWorkspaceNavigation();
  return [
    {
      label: "Deploy your first app",
      hint: "GitHub or image",
      href: routes.projects.list({ workspaceSlug: workspace.slug, new: true }),
    },
    {
      label: "Create a keyspace",
      hint: "API keys",
      href: routes.apis.list({ workspaceSlug: workspace.slug, new: true }),
    },
    {
      label: "Set up with your agent",
      hint: "MCP",
      href: routes.projects.list({ workspaceSlug: workspace.slug }),
    },
  ];
}

export function GetStarted({ density }: { density: VariantProps["options"]["density"] }) {
  const actions = useGetStartedActions();
  return (
    <div className="flex flex-col">
      {actions.map((action) => (
        <ActionRow key={action.label} action={action} density={density} />
      ))}
    </div>
  );
}

export function DeployNudge({ compact = false }: { compact?: boolean }) {
  const state = useDeployState();
  const shell = cn(
    "group flex items-center gap-2.5 rounded-lg border px-3 text-[13px] transition-colors",
    compact ? "h-9" : "h-11",
  );

  if (state.kind === "loading") {
    return <div className={cn(shell, "border-grayA-4")} aria-busy="true" />;
  }

  if (state.kind === "none") {
    return (
      <Link
        href={state.href}
        className={cn(shell, "border-dashed border-grayA-6 hover:border-grayA-8 hover:bg-grayA-2")}
      >
        <IconBoltOutline18 className="size-3.5 shrink-0 text-gray-9" />
        <span className="min-w-0 flex-1 truncate text-gray-11">No apps deployed yet</span>
        <span className="shrink-0 text-xs text-accent-12 group-hover:underline">Deploy</span>
      </Link>
    );
  }

  return (
    <Link
      href={state.href}
      className={cn(shell, "border-grayA-4 hover:border-grayA-7 hover:bg-grayA-2")}
    >
      <span
        className={cn(
          "size-1.5 shrink-0 rounded-full",
          state.inFlight ? "bg-warning-9" : "bg-success-9",
        )}
      />
      <span className="flex min-w-0 flex-1 flex-col leading-tight">
        <span className="truncate text-accent-12">{state.appName}</span>
        <span className="truncate text-xs text-gray-9">{state.projectName}</span>
      </span>
      <TimestampInfo value={state.deployedAt} className="shrink-0 text-xs text-gray-9" />
    </Link>
  );
}

export function RowMeta({
  row,
  showProject,
  className,
}: {
  row: LaunchpadRow;
  showProject: boolean;
  className?: string;
}) {
  const parts: string[] = [];
  if (row.kind === "keyspace" && row.keyCount > 0) {
    parts.push(`${fmt(row.keyCount)} keys`);
  }
  if (row.kind === "ratelimit" && row.total > 0) {
    parts.push(`${Math.round((row.failed / row.total) * 1000) / 10}% blocked`);
  }
  const project = showProject ? row.projectName : null;
  if (parts.length === 0 && !project) {
    return null;
  }
  return (
    <span className={cn("flex min-w-0 items-center gap-1 text-xs text-gray-9", className)}>
      {parts.length > 0 && <span className="truncate">{parts.join(" · ")}</span>}
      {parts.length > 0 && project && <span className="text-gray-7">·</span>}
      {project && <ProjectChip name={project} />}
    </span>
  );
}

export function Value({ row, className }: { row: LaunchpadRow; className?: string }) {
  return (
    <span className={cn("shrink-0 tabular-nums text-accent-12", className)}>{fmt(row.total)}</span>
  );
}

export function RowSpark({ row, options }: { row: LaunchpadRow; options: VariantProps["options"] }) {
  return <Spark buckets={row.buckets} mode={options.spark} />;
}

export function KindIcon({ row }: { row: LaunchpadRow }) {
  const Glyph = row.kind === "keyspace" ? IconKey2Outline18 : IconCubeOutline18;
  return <Glyph className="size-3.5 shrink-0 text-gray-9" />;
}
