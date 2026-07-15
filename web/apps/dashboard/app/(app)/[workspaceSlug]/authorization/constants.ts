import { routes } from "@/lib/navigation/routes";

export const navigation = (workspaceSlug: string) => [
  {
    label: "Roles",
    href: routes.authorization.roles({ workspaceSlug }),
    segment: "roles",
  },
  {
    label: "Permissions",
    href: routes.authorization.permissions({ workspaceSlug }),
    segment: "permissions",
  },
];
