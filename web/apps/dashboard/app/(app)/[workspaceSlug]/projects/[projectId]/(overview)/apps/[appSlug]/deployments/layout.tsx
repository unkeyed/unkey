"use client";

import { V2B_HEADER_HEIGHT } from "@/components/navigation/variants/shared/v2b-header";
import {
  type V2bDeploymentsVariant,
  useV2bDeploymentsVariant,
} from "@/hooks/use-v2b-deployments-variant";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { generateFakeDeployments } from "@/lib/fake-deployments";
import { shortenId } from "@/lib/shorten-id";
import { cn } from "@/lib/utils";
import { XMark } from "@unkey/icons";
import Link from "next/link";
import { useParams, usePathname, useRouter } from "next/navigation";
import { type PropsWithChildren, useEffect } from "react";
import { ProjectContentWrapper } from "../../../../components/project-content-wrapper";
import { DeploymentsListControls } from "../../../deployments/components/controls";
import { DeploymentsCardList } from "../../../deployments/components/deployments-card-list";
import { DeploymentsHeader } from "../../../deployments/components/deployments-header";
import { DeploymentStatusBadge } from "../../../../components/deployment-status-badge";

/**
 * App → Deployments shell. Picks one of six layouts based on the
 * v2b sub-variant:
 *   a · rail+crumb         (default; rail + in-page crumb)
 *   b · railless           (no rail, in-page crumb only)
 *   c · merged-crumb       (rail; deployment crumb merged into top header)
 *   d · eyebrow            (no rail; detail uses ← back + page H1)
 *   e · drawer             (no rail; detail slides in over the still-mounted list)
 *   f · split              (no rail; on detail, list compresses to left rail)
 */
export default function AppDeploymentsLayout({ children }: PropsWithChildren) {
  const { variant } = useV2bDeploymentsVariant();
  const params = useParams();
  const pathname = usePathname() ?? "";
  const appSlug = typeof params?.appSlug === "string" ? params.appSlug : "";
  const detailMatch = pathname.match(
    new RegExp(`/apps/${appSlug}/deployments/([^/]+)$`),
  );
  const isDetail = Boolean(detailMatch);

  if (variant === "e" && isDetail) {
    return <DrawerShell>{children}</DrawerShell>;
  }
  if (variant === "f" && isDetail) {
    return <SplitShell currentDeploymentId={detailMatch?.[1] ?? ""}>{children}</SplitShell>;
  }

  return (
    <DefaultShell variant={variant}>{children}</DefaultShell>
  );
}

function DefaultShell({
  variant,
  children,
}: PropsWithChildren<{ variant: V2bDeploymentsVariant }>) {
  const showRail = variant === "a" || variant === "c";
  return (
    <div
      className="flex w-full flex-1"
      style={{ minHeight: `calc(100svh - ${V2B_HEADER_HEIGHT}px)` }}
    >
      {showRail ? <ManageSidebar /> : null}
      <div className="flex flex-1 flex-col overflow-auto">{children}</div>
    </div>
  );
}

function ManageSidebar() {
  const workspace = useWorkspaceNavigation();
  const params = useParams();
  const pathname = usePathname() ?? "";
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";
  const appSlug = typeof params?.appSlug === "string" ? params.appSlug : "";

  const deploymentsHref = `/${workspace.slug}/projects/${projectId}/apps/${appSlug}/deployments`;
  const inDeployments = pathname.includes(`/apps/${appSlug}/deployments`);

  return (
    <aside className="flex w-52 shrink-0 flex-col self-stretch border-r border-grayA-4 px-2 py-2">
      <div className="px-2 pb-1.5 pt-1 text-[11px] font-medium uppercase tracking-wide text-gray-11">
        Manage
      </div>
      <nav className="flex flex-col gap-1">
        <Link
          href={deploymentsHref}
          className={cn(
            "flex items-center gap-2 rounded-md px-2 py-1.5 text-[13px] font-medium transition-colors",
            inDeployments
              ? "bg-grayA-3 text-accent-12"
              : "text-gray-11 hover:bg-grayA-3 hover:text-accent-12",
          )}
        >
          Deployments
        </Link>
      </nav>
    </aside>
  );
}

/**
 * Drawer (sub-variant `e`): the deployment list stays visible behind a
 * right-side sheet that holds the detail. Backdrop click and ESC return
 * to the list URL — the detail mounts/unmounts via routing as usual.
 */
function DrawerShell({ children }: PropsWithChildren) {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const params = useParams();
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";
  const appSlug = typeof params?.appSlug === "string" ? params.appSlug : "";
  const listHref = `/${workspace.slug}/projects/${projectId}/apps/${appSlug}/deployments`;

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        router.push(listHref);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [router, listHref]);

  return (
    <div
      className="relative flex w-full flex-1 flex-col overflow-auto"
      style={{ minHeight: `calc(100svh - ${V2B_HEADER_HEIGHT}px)` }}
    >
      <ProjectContentWrapper centered className="pt-12">
        <DeploymentsHeader />
        <DeploymentsListControls />
        <DeploymentsCardList />
      </ProjectContentWrapper>
      <button
        type="button"
        aria-label="Close"
        onClick={() => router.push(listHref)}
        className="fixed inset-x-0 z-40 bg-black/40"
        style={{ top: V2B_HEADER_HEIGHT, bottom: 0 }}
      />
      <aside
        className="fixed right-0 z-40 flex w-[clamp(640px,80vw,1100px)] flex-col bg-background shadow-2xl"
        style={{ top: V2B_HEADER_HEIGHT, bottom: 0 }}
        role="dialog"
        aria-modal="true"
      >
        <div className="flex h-12 shrink-0 items-center justify-between border-b border-grayA-4 px-4">
          <Link
            href={listHref}
            className="text-[13px] font-medium text-gray-11 hover:text-accent-12"
          >
            ← Back to deployments
          </Link>
          <button
            type="button"
            aria-label="Close"
            onClick={() => router.push(listHref)}
            className="flex size-8 items-center justify-center rounded-md text-gray-11 hover:bg-grayA-3 hover:text-accent-12"
          >
            <XMark iconSize="md-regular" />
          </button>
        </div>
        <div className="flex flex-1 flex-col overflow-auto">{children}</div>
      </aside>
    </div>
  );
}

/**
 * Split (sub-variant `f`): on a detail URL the list compresses to a
 * left-side rail of compact rows; the detail occupies the right pane.
 * On the list URL itself, no rail — the list takes the full width via
 * the default shell (handled upstream).
 */
function SplitShell({
  children,
  currentDeploymentId,
}: PropsWithChildren<{ currentDeploymentId: string }>) {
  return (
    <div
      className="flex w-full flex-1"
      style={{ minHeight: `calc(100svh - ${V2B_HEADER_HEIGHT}px)` }}
    >
      <DeploymentsRail currentDeploymentId={currentDeploymentId} />
      <div className="flex flex-1 flex-col overflow-auto">{children}</div>
    </div>
  );
}

function DeploymentsRail({ currentDeploymentId }: { currentDeploymentId: string }) {
  const workspace = useWorkspaceNavigation();
  const params = useParams();
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";
  const appSlug = typeof params?.appSlug === "string" ? params.appSlug : "";
  const listHref = `/${workspace.slug}/projects/${projectId}/apps/${appSlug}/deployments`;
  // Prototype-only: rail uses the same fake set as the list to feel populated.
  const items = generateFakeDeployments(projectId);

  return (
    <aside className="flex w-72 shrink-0 flex-col self-stretch border-r border-grayA-4">
      <div className="flex items-center justify-between border-b border-grayA-4 px-3 py-2.5">
        <Link
          href={listHref}
          className="text-[13px] font-semibold text-accent-12 hover:underline"
        >
          Deployments
        </Link>
      </div>
      <nav className="flex flex-1 flex-col overflow-auto">
        {items.map((d) => {
          const active = d.id === currentDeploymentId;
          return (
            <Link
              key={d.id}
              href={`${listHref}/${d.id}`}
              className={cn(
                "flex flex-col gap-0.5 border-b border-grayA-3 px-3 py-2.5 text-[12px] transition-colors",
                active ? "bg-grayA-3" : "hover:bg-grayA-2",
              )}
            >
              <div className="flex items-center justify-between gap-2">
                <span className="truncate font-mono font-medium text-accent-12">
                  {shortenId(d.id)}
                </span>
                <DeploymentStatusBadge status={d.status} />
              </div>
              <span className="truncate text-gray-11">{d.gitCommitMessage}</span>
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
