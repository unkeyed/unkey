"use client";

import { Logomark } from "@/components/logomark";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useProjectItems } from "@/hooks/use-project-items";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { setLastUsedOrgCookie, setSessionCookie } from "@/lib/auth/cookies";
import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { useLiveQuery } from "@tanstack/react-db";
import { ChevronExpandY, Cube, Plus } from "@unkey/icons";
import { toast } from "@unkey/ui";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useMemo } from "react";
import { HelpButton } from "../../sidebar/help-button";
import { UserButton } from "../../sidebar/user-button";
import { CrumbPopover, type CrumbPopoverFooter, type CrumbPopoverItem } from "./crumb-popover";

export const V2B_HEADER_HEIGHT = 52;

/**
 * v2b — full-width top header with clickable breadcrumb crumbs.
 *
 * Supabase-inspired: each crumb is a label linking to the target +
 * a small chevron-updown button that opens a popover to switch to a
 * sibling or take a contextual action. Right side carries the Help menu
 * and the User avatar menu.
 */
export function V2BTopHeader() {
  const workspace = useWorkspaceNavigation();
  const pathname = usePathname() ?? "";
  const base = `/${workspace.slug}`;

  const match = pathname.match(/\/projects\/([^/]+)(?:\/apps\/([^/]+))?/);
  const projectId = match?.[1];
  const appSlug = match?.[2];

  return (
    <header
      className="fixed inset-x-0 top-0 z-30 w-full border-b border-grayA-4 bg-gray-1"
      style={{ height: V2B_HEADER_HEIGHT }}
    >
      <div className="flex h-full w-full items-center gap-1 px-4">
        <Link href={base} aria-label="Unkey" className="inline-flex items-center">
          <Logomark />
        </Link>
        <Slash />
        <WorkspaceCrumb />
        {projectId ? (
          <>
            <Slash />
            <ProjectCrumb projectId={projectId} />
          </>
        ) : null}
        {projectId && appSlug ? (
          <>
            <Slash />
            <AppCrumb projectId={projectId} appSlug={appSlug} />
          </>
        ) : null}
        <div className="ml-auto flex items-center gap-1">
          <HelpButton />
          <UserButton />
        </div>
      </div>
    </header>
  );
}

function Slash() {
  return <span className="select-none px-0.5 text-gray-7">/</span>;
}

type CrumbProps = {
  icon: React.ReactNode;
  label: string;
  href: string;
  items: CrumbPopoverItem[];
  currentId: string;
  searchPlaceholder: string;
  emptyText: string;
  footer: CrumbPopoverFooter;
};

/**
 * A single crumb: clickable label linking to its target + a
 * chevron-updown button that opens a switcher popover.
 */
function Crumb({
  icon,
  label,
  href,
  items,
  currentId,
  searchPlaceholder,
  emptyText,
  footer,
}: CrumbProps) {
  return (
    <div className="flex items-center gap-0.5">
      <Link
        href={href}
        className="flex items-center gap-1.5 px-1 py-1 text-[13px] font-medium text-accent-12"
      >
        {icon}
        <span className="truncate max-w-[180px]">{label}</span>
      </Link>
      <CrumbPopover
        items={items}
        currentId={currentId}
        searchPlaceholder={searchPlaceholder}
        emptyText={emptyText}
        footer={footer}
      >
        <button
          type="button"
          className="flex size-6 items-center justify-center rounded-md text-gray-11 hover:bg-grayA-3 hover:text-accent-12"
          aria-label={`Switch ${label}`}
        >
          <ChevronExpandY className="size-3" iconSize="sm-regular" />
        </button>
      </CrumbPopover>
    </div>
  );
}

function WorkspaceCrumb() {
  const workspace = useWorkspaceNavigation();
  const { data: user } = trpc.user.getCurrentUser.useQuery();
  const { data: memberships } = trpc.user.listMemberships.useQuery(user?.id as string, {
    enabled: !!user,
  });
  const orgs = memberships?.data ?? [];

  const switchOrg = trpc.user.switchOrg.useMutation({
    async onSuccess(sessionData, orgId) {
      if (!sessionData.token || !sessionData.expiresAt) {
        toast.error("Failed to switch workspace. Invalid session data.");
        return;
      }
      try {
        await setSessionCookie({
          token: sessionData.token,
          expiresAt: sessionData.expiresAt,
        });
      } catch {
        toast.error("Failed to complete workspace switch. Please try again.");
        return;
      }
      try {
        await setLastUsedOrgCookie({ orgId });
      } catch {
        // Non-critical; storage failure shouldn't block the switch.
      }
      window.location.replace("/");
    },
    onError() {
      toast.error("Failed to switch workspace. Contact support if error persists.");
    },
  });

  const items: CrumbPopoverItem[] = useMemo(
    () =>
      orgs.map((m) => ({
        id: m.organization.id,
        label: m.organization.name,
        onClick: () => {
          if (m.organization.id !== workspace.orgId) {
            switchOrg.mutate(m.organization.id);
          }
        },
      })),
    [orgs, switchOrg, workspace.orgId],
  );

  return (
    <Crumb
      icon={
        <Avatar className="size-4 rounded-sm border border-grayA-6 shrink-0">
          <AvatarFallback name={workspace.name} variant="marble" square />
        </Avatar>
      }
      label={workspace.name}
      href={`/${workspace.slug}`}
      items={items}
      currentId={workspace.orgId}
      searchPlaceholder="Find workspace..."
      emptyText="No workspaces found"
      footer={{ icon: Plus, label: "New workspace", href: "/new" }}
    />
  );
}

function ProjectCrumb({ projectId }: { projectId: string }) {
  const workspace = useWorkspaceNavigation();
  const projectsQuery = useLiveQuery((q) =>
    q.from({ project: collection.projects }).select(({ project }) => ({
      id: project.id,
      name: project.name,
    })),
  );
  const projects = projectsQuery.data ?? [];
  const current = projects.find((p) => p.id === projectId);

  const items: CrumbPopoverItem[] = projects.map((p) => ({
    id: p.id,
    label: p.name,
    href: `/${workspace.slug}/projects/${p.id}`,
  }));

  return (
    <Crumb
      icon={<Cube className="size-3.5 text-accent-11" iconSize="sm-regular" />}
      label={current?.name ?? projectId}
      href={`/${workspace.slug}/projects/${projectId}`}
      items={items}
      currentId={projectId}
      searchPlaceholder="Find project..."
      emptyText="No projects found"
      footer={{
        icon: Plus,
        label: "New project",
        href: `/${workspace.slug}/projects/new`,
      }}
    />
  );
}

function AppCrumb({ projectId, appSlug }: { projectId: string; appSlug: string }) {
  const workspace = useWorkspaceNavigation();
  const { items: projectItems } = useProjectItems(projectId);
  const apps = projectItems.filter((i) => i.type === "app");
  const current = apps.find((a) => a.slug === appSlug);

  const items: CrumbPopoverItem[] = apps.map((a) => ({
    id: a.slug,
    label: a.name,
    href: `/${workspace.slug}/projects/${projectId}/apps/${a.slug}`,
  }));

  return (
    <Crumb
      icon={<Cube className="size-3.5 text-accent-11" iconSize="sm-regular" />}
      label={current?.name ?? appSlug}
      href={`/${workspace.slug}/projects/${projectId}/apps/${appSlug}`}
      items={items}
      currentId={appSlug}
      searchPlaceholder="Find app..."
      emptyText="No apps found"
      footer={{
        icon: Plus,
        label: "New app",
        href: `/${workspace.slug}/projects/${projectId}`,
      }}
    />
  );
}
