import { routes } from "@/lib/navigation/routes";

export type AuthorizationScope = { workspaceSlug: string; projectId?: string };

const ITEMS = [
  {
    segment: "roles",
    label: "Roles",
    workspace: routes.authorization.roles,
    project: routes.projects.authorizationRoles,
  },
  {
    segment: "permissions",
    label: "Permissions",
    workspace: routes.authorization.permissions,
    project: routes.projects.authorizationPermissions,
  },
] as const;

export const navigation = ({ workspaceSlug, projectId }: AuthorizationScope) =>
  ITEMS.map((item) => ({
    label: item.label,
    segment: item.segment,
    href: projectId
      ? item.project({ workspaceSlug, projectId })
      : item.workspace({ workspaceSlug }),
  }));
