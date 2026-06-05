"use client";

import { getButtonStyles } from "@/components/navigation/sidebar/app-sidebar/components/nav-items/utils";
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { useApiKeyAuthId } from "@/hooks/use-api-key-auth-id";
import { useSectionContext } from "@/hooks/use-section-context";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useFlag } from "@/lib/flags/provider";
import {
  buildApiLinks,
  buildAuthorizationLinks,
  buildDeploymentLinks,
  buildKeyDetailLinks,
  buildNamespaceLinks,
  buildProjectLinks,
  buildSettingsLinks,
  buildWorkspaceSections,
} from "@/lib/navigation/leaves";
import type { SidebarAction } from "@/lib/navigation/types";
import type { SectionContext } from "@/hooks/use-section-context";
import { ChevronLeft } from "@unkey/icons";
import Link from "next/link";
import { useSelectedLayoutSegments } from "next/navigation";
import { NavLinkList } from "./nav-link-list";
import { useSidebarActionsState } from "./sidebar-actions-context";

function getUpTarget(
  context: SectionContext,
  slug: string,
): { label: string; href: string } | null {
  switch (context.type) {
    case "api":
      return { label: "Keyspaces (APIs)", href: `/${slug}/apis` };
    case "keyDetail":
      return {
        label: "Keys",
        href: `/${slug}/apis/${context.apiId}/keys/${context.keyAuthId}`,
      };
    case "project":
      return { label: "Projects", href: `/${slug}/projects` };
    case "deployment":
      return {
        label: "Deployments",
        href: `/${slug}/projects/${context.projectId}/apps/${context.appId}/deployments`,
      };
    case "namespace":
      return { label: "Ratelimits", href: `/${slug}/ratelimits` };
    case "identity":
      return { label: "Identities", href: `/${slug}/identities` };
    case "settings":
    case "authorization":
      return { label: "Workspace", href: `/${slug}` };
    default:
      return null;
  }
}

export function SidebarBody() {
  const contextualNav = useFlag("contextualNav");
  const context = useSectionContext();
  const segments = useSelectedLayoutSegments().slice(1);
  const { slug } = useWorkspaceNavigation();
  const keyAuthId = useApiKeyAuthId(context.type === "api" ? context.apiId : undefined);
  const dynamicActions = useSidebarActionsState();

  const links = (() => {
    switch (context.type) {
      case "workspace":
      case "identity":
        return buildWorkspaceSections(slug, segments);
      case "settings":
        return buildSettingsLinks(slug, segments);
      case "authorization":
        return buildAuthorizationLinks(slug, segments);
      case "project":
        return buildProjectLinks(slug, context.projectId, segments);
      case "deployment":
        return buildDeploymentLinks(
          slug,
          context.projectId,
          context.appId,
          context.deploymentId,
          segments,
        );
      case "api":
        return buildApiLinks(slug, context.apiId, keyAuthId, segments);
      case "keyDetail":
        return buildKeyDetailLinks(
          slug,
          context.apiId,
          context.keyAuthId,
          context.keyId,
          segments,
        );
      case "namespace":
        return buildNamespaceLinks(slug, context.namespaceId, segments);
    }
  })();

  if (!contextualNav) {
    return <NavLinkList links={links} />;
  }

  const up = getUpTarget(context, slug);

  return (
    <>
      {up ? <UpRow label={up.label} href={up.href} /> : null}
      <NavLinkList links={links} />
      {dynamicActions.length > 0 ? <ActionList actions={dynamicActions} /> : null}
    </>
  );
}

function UpRow({ label, href }: { label: string; href: string }) {
  return (
    <SidebarGroup>
      <SidebarGroupContent>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton asChild tooltip={label} className={getButtonStyles(false)}>
              <Link href={href}>
                <ChevronLeft iconSize="xl-medium" />
                <span>{label}</span>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}

function ActionList({ actions }: { actions: SidebarAction[] }) {
  return (
    <SidebarGroup>
      <SidebarGroupLabel>Actions</SidebarGroupLabel>
      <SidebarGroupContent>
        <SidebarMenu>
          {actions.map((action) => (
            <ActionRow key={action.key} action={action} />
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}

function ActionRow({ action }: { action: SidebarAction }) {
  const Icon = action.icon;
  const contents = (
    <>
      {Icon ? <Icon iconSize="xl-medium" /> : null}
      <span>{action.label}</span>
    </>
  );

  if (action.href && !action.disabled) {
    return (
      <SidebarMenuItem>
        <SidebarMenuButton asChild tooltip={action.label} className={getButtonStyles(false)}>
          <Link href={action.href}>{contents}</Link>
        </SidebarMenuButton>
      </SidebarMenuItem>
    );
  }

  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        tooltip={action.label}
        disabled={action.disabled}
        onClick={action.onClick}
        className={getButtonStyles(false)}
      >
        {contents}
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}
