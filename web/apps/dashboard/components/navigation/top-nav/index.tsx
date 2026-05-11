"use client";

import { Logomark } from "@/components/logomark";
import { useSidebar } from "@/components/ui/sidebar";
import { type BreadcrumbDescriptor, useBreadcrumbs } from "@/hooks/use-breadcrumbs";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Menu } from "@unkey/icons";
import Link from "next/link";
import { Fragment } from "react";
import { HelpButton } from "../sidebar/help-button";
import { UserButton } from "../sidebar/user-button";
import { ApiCrumb } from "./api-crumb";
import { CrumbSeparator } from "./crumb";
import { TopNavFeedbackButton } from "./feedback-button";
import { NamespaceCrumb } from "./namespace-crumb";
import { ProjectCrumb } from "./project-crumb";
import { WorkspaceCrumb } from "./workspace-crumb";

export const TOP_NAV_HEIGHT = 52;

export function TopNav() {
  const workspace = useWorkspaceNavigation();
  const crumbs = useBreadcrumbs();
  // The shadcn Sidebar primitive renders itself as a Sheet on mobile via
  // useSidebar().setOpenMobile — reuse that instead of mounting a parallel
  // drawer, otherwise the two stack on top of each other.
  const { setOpenMobile } = useSidebar();

  return (
    <header
      className="z-30 flex w-full shrink-0 items-center gap-1 border-b border-grayA-4 bg-gray-1 px-4"
      style={{ height: TOP_NAV_HEIGHT }}
    >
      <Link href={`/${workspace.slug}`} aria-label="Unkey" className="inline-flex items-center">
        <Logomark />
      </Link>
      {/* Crumbs render on every viewport. min-w-0 lets the labels
          truncate when the chain would otherwise overflow on narrow
          screens. Chevron switchers are hidden on mobile inside the
          Crumb primitive itself. */}
      <div className="flex min-w-0 items-center gap-1">
        <CrumbSeparator />
        {crumbs.map((descriptor, i) => (
          <Fragment key={crumbKey(descriptor)}>
            {i > 0 && <CrumbSeparator />}
            <CrumbForDescriptor descriptor={descriptor} />
          </Fragment>
        ))}
      </div>
      <div className="ml-auto flex shrink-0 items-center gap-1">
        <TopNavFeedbackButton className="hidden md:inline-flex" />
        <HelpButton />
        <UserButton />
        <button
          type="button"
          onClick={() => setOpenMobile(true)}
          aria-label="Open navigation"
          className="flex size-8 items-center justify-center rounded-md text-gray-11 hover:bg-grayA-3 hover:text-accent-12 md:hidden"
        >
          <Menu className="size-4" iconSize="md-regular" />
        </button>
      </div>
    </header>
  );
}

function CrumbForDescriptor({ descriptor }: { descriptor: BreadcrumbDescriptor }) {
  switch (descriptor.type) {
    case "workspace":
      return <WorkspaceCrumb />;
    case "project":
      return <ProjectCrumb projectId={descriptor.projectId} />;
    case "api":
      return <ApiCrumb apiId={descriptor.apiId} />;
    case "namespace":
      return <NamespaceCrumb namespaceId={descriptor.namespaceId} />;
  }
}

function crumbKey(descriptor: BreadcrumbDescriptor): string {
  switch (descriptor.type) {
    case "workspace":
      return "workspace";
    case "project":
      return `project:${descriptor.projectId}`;
    case "api":
      return `api:${descriptor.apiId}`;
    case "namespace":
      return `namespace:${descriptor.namespaceId}`;
  }
}
