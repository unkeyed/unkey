"use client";

import type { UnkeyPermission } from "@unkey/rbac";

export type ScopedItem = {
  id: string;
  name: string;
};

export type ApiItem = ScopedItem & {
  keyspaceId: string | null;
};

export type PermissionResourceSuggestions = {
  projects: ScopedItem[];
  apps: Array<ScopedItem & { projectId: string }>;
  environments: Array<ScopedItem & { projectId: string; appId: string }>;
  deployments: Array<ScopedItem & { projectId: string; appId: string; environmentId: string }>;
  ratelimitNamespaces: ScopedItem[];
  ratelimitOverrides: Array<ScopedItem & { namespaceId: string }>;
  roles: ScopedItem[];
  permissions: ScopedItem[];
  identities: ScopedItem[];
};

export type UrnPermissionVariantProps = {
  workspaceId: string;
  apis: ApiItem[];
  projects: ScopedItem[];
  permissionResources?: PermissionResourceSuggestions;
  selectedPermissions: UnkeyPermission[];
  onChange: (permissions: UnkeyPermission[]) => void;
  loadMore?: () => void;
  hasNextPage?: boolean;
  isFetchingNextPage?: boolean;
  onClose: () => void;
};

export type ActionGroupID =
  | "keys"
  | "ratelimits"
  | "authorization"
  | "identities"
  | "deployments"
  | "admin";

export type ActionDefinition = {
  id: string;
  label: string;
  description: string;
  group: ActionGroupID;
  category: string;
  categoryLabel: string;
  terminalLabel: string;
};
