import { routes } from "@/lib/navigation/routes";
import type { ResourceScope } from "@/lib/navigation/routes/shared";

const ITEMS = [
  { segment: "roles", label: "Roles", getHref: routes.authorization.roles },
  { segment: "permissions", label: "Permissions", getHref: routes.authorization.permissions },
] as const;

export const navigation = (scope: ResourceScope) =>
  ITEMS.map((item) => ({ label: item.label, segment: item.segment, href: item.getHref(scope) }));
