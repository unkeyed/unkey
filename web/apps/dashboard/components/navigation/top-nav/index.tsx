"use client";

import { Logomark } from "@/components/logomark";
import { useSidebar } from "@/components/ui/sidebar";
import {
  type BreadcrumbDescriptor,
  useBreadcrumbs,
  useWorkspaceCrumbHref,
} from "@/hooks/use-breadcrumbs";
import { useHydrated } from "@/hooks/use-hydrated";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { IconMenuOutline18 } from "@unkey/icons";
import Link from "next/link";
import { Fragment } from "react";
import { AccountCrumb } from "./account-crumb";
import { ApiCrumb } from "./api-crumb";
import { AppCrumb } from "./app-crumb";
import { CrumbSeparator } from "./crumb";
import { TopNavFeedbackButton } from "./feedback-button";
import { HelpButton } from "./help-button";
import { IdentityCrumb } from "./identity-crumb";
import { NamespaceCrumb } from "./namespace-crumb";
import { ProjectCrumb } from "./project-crumb";
import { UserButton } from "./user-button";
import { WorkspaceCrumb } from "./workspace-crumb";

export function TopNav() {
  const workspace = useWorkspaceNavigation();
  const workspaceCrumbHref = useWorkspaceCrumbHref();
  const hydrated = useHydrated();
  const { setOpenMobile } = useSidebar();

  return (
    <header className="flex h-(--app-nav-h) w-full shrink-0 items-center gap-1 border-b bg-background px-4">
      <Link
        href={routes.workspaces.overview({ workspaceSlug: workspace.slug })}
        aria-label="Unkey"
        className="inline-flex items-center"
      >
        <Logomark />
      </Link>
      <div className="flex min-w-0 items-center gap-1">
        <CrumbSeparator />
        <WorkspaceCrumb href={workspaceCrumbHref} />
        {/* The trail resolves resource owners with live queries, which cannot render on the server. */}
        {hydrated ? <TrailingCrumbs /> : null}
      </div>
      <div className="ml-auto flex shrink-0 items-center gap-2">
        <TopNavFeedbackButton className="hidden md:flex" />
        <HelpButton />
        <UserButton />
        <button
          type="button"
          onClick={() => setOpenMobile(true)}
          aria-label="Open navigation"
          className="flex size-8 items-center justify-center rounded-md text-gray-11 hover:bg-grayA-3 hover:text-gray-12 md:hidden"
        >
          <IconMenuOutline18 className="size-4" />
        </button>
      </div>
    </header>
  );
}

function TrailingCrumbs() {
  const crumbs = useBreadcrumbs();
  return crumbs.slice(1).map((descriptor) => (
    <Fragment key={crumbKey(descriptor)}>
      <CrumbSeparator />
      <CrumbForDescriptor descriptor={descriptor} />
    </Fragment>
  ));
}

function CrumbForDescriptor({ descriptor }: { descriptor: BreadcrumbDescriptor }) {
  switch (descriptor.type) {
    case "workspace":
      return <WorkspaceCrumb href={descriptor.href} />;
    case "account":
      return <AccountCrumb />;
    case "project":
      return <ProjectCrumb owner={descriptor.owner} />;
    case "app":
      return <AppCrumb projectId={descriptor.projectId} appId={descriptor.appId} />;
    case "api":
      return <ApiCrumb apiId={descriptor.apiId} projectId={descriptor.projectId} />;
    case "namespace":
      return (
        <NamespaceCrumb namespaceId={descriptor.namespaceId} projectId={descriptor.projectId} />
      );
    case "identity":
      return <IdentityCrumb identityId={descriptor.identityId} projectId={descriptor.projectId} />;
  }
}

function crumbKey(descriptor: BreadcrumbDescriptor): string {
  switch (descriptor.type) {
    case "workspace":
      return "workspace";
    case "account":
      return "account";
    // One project crumb at most, and a stable key keeps it mounted while the
    // owning-project lookup resolves.
    case "project":
      return "project";
    case "app":
      return `app:${descriptor.appId}`;
    case "api":
      return `api:${descriptor.apiId}`;
    case "namespace":
      return `namespace:${descriptor.namespaceId}`;
    case "identity":
      return `identity:${descriptor.identityId}`;
  }
}
