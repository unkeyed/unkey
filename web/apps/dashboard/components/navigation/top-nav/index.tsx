"use client";

import { useDeploymentNavVariant } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/deployments/[deploymentId]/use-deployment-nav-variant";
import { Logomark } from "@/components/logomark";
import { useSidebar } from "@/components/ui/sidebar";
import { type BreadcrumbDescriptor, useBreadcrumbs } from "@/hooks/use-breadcrumbs";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Menu } from "@unkey/icons";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Fragment } from "react";
import { HelpButton } from "../sidebar/help-button";
import { UserButton } from "../sidebar/user-button";
import { ApiCrumb } from "./api-crumb";
import { AppCrumb } from "./app-crumb";
import { CrumbSeparator } from "./crumb";
import { DeploymentCrumb } from "./deployment-crumb";
import { TopNavFeedbackButton } from "./feedback-button";
import { IdentityCrumb } from "./identity-crumb";
import { NamespaceCrumb } from "./namespace-crumb";
import { ProjectCrumb } from "./project-crumb";
import { WorkspaceCrumb } from "./workspace-crumb";

export const TOP_NAV_HEIGHT = 52;

export function TopNav() {
  const workspace = useWorkspaceNavigation();
  const crumbs = useBreadcrumbs();
  const { setOpenMobile } = useSidebar();
  const [navVariant] = useDeploymentNavVariant();
  const params = useParams<{ projectId?: string; appId?: string; deploymentId?: string }>();
  // In the "crumb" variant the deployment becomes a fourth top-bar crumb.
  const deploymentCrumb =
    navVariant === "crumb" && params.projectId && params.appId && params.deploymentId
      ? {
          deploymentId: params.deploymentId,
          href: `/${workspace.slug}/projects/${params.projectId}/apps/${params.appId}/deployments/${params.deploymentId}`,
        }
      : null;

  return (
    <header
      className="flex w-full shrink-0 items-center gap-1 border-b border-grayA-4 bg-gray-1 px-4"
      style={{ height: TOP_NAV_HEIGHT }}
    >
      <Link href={`/${workspace.slug}`} aria-label="Unkey" className="inline-flex items-center">
        <Logomark />
      </Link>
      <div className="flex min-w-0 items-center gap-1">
        <CrumbSeparator />
        {crumbs.map((descriptor, i) => (
          <Fragment key={crumbKey(descriptor)}>
            {i > 0 && <CrumbSeparator />}
            <CrumbForDescriptor descriptor={descriptor} />
          </Fragment>
        ))}
        {deploymentCrumb && (
          <>
            <CrumbSeparator />
            <DeploymentCrumb
              href={deploymentCrumb.href}
              deploymentId={deploymentCrumb.deploymentId}
            />
          </>
        )}
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
      return <WorkspaceCrumb href={descriptor.href} />;
    case "project":
      return <ProjectCrumb projectId={descriptor.projectId} />;
    case "app":
      return <AppCrumb projectId={descriptor.projectId} appId={descriptor.appId} />;
    case "api":
      return <ApiCrumb apiId={descriptor.apiId} />;
    case "namespace":
      return <NamespaceCrumb namespaceId={descriptor.namespaceId} />;
    case "identity":
      return <IdentityCrumb identityId={descriptor.identityId} />;
  }
}

function crumbKey(descriptor: BreadcrumbDescriptor): string {
  switch (descriptor.type) {
    case "workspace":
      return "workspace";
    case "project":
      return `project:${descriptor.projectId}`;
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
