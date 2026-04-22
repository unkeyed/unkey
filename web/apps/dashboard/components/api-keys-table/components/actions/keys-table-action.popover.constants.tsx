import { MAX_KEYS_FETCH_LIMIT } from "@/app/(app)/[workspaceSlug]/authorization/roles/components/upsert-role/components/assign-key/hooks/use-fetch-keys";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { trpc } from "@/lib/trpc/client";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import {
  ArrowDottedRotateAnticlockwise,
  ArrowOppositeDirectionY,
  Ban,
  CalendarClock,
  ChartPie,
  Check,
  Clone,
  Code,
  Gauge,
  PenWriting3,
  Tag,
  Trash,
} from "@unkey/icons";
import { toast } from "@unkey/ui";
import { DeleteKey } from "./components/delete-key";
import { UpdateKeyStatus } from "./components/disable-key";
import { EditCredits } from "./components/edit-credits";
import { EditExpiration } from "./components/edit-expiration";
import { EditExternalId } from "./components/edit-external-id";
import { EditKeyName } from "./components/edit-key-name";
import { EditMetadata } from "./components/edit-metadata";
import { EditRatelimits } from "./components/edit-ratelimits";
import { KeyRbacDialog } from "./components/edit-rbac";
import { MAX_PERMS_FETCH_LIMIT } from "./components/edit-rbac/components/assign-permission/hooks/use-fetch-keys-permissions";
import { MAX_ROLES_FETCH_LIMIT } from "./components/edit-rbac/components/assign-role/hooks/use-fetch-keys-roles";
import { RotateKey } from "./components/rotate-key/rotate-key";

type KeyContext = {
  apiId?: string;
  keyspaceId?: string | null;
};

export const getKeysTableActionItems = (
  key: KeyDetails,
  trpcUtils: ReturnType<typeof trpc.useUtils>,
  context: KeyContext = {},
): MenuItem[] => {
  const { apiId, keyspaceId } = context;
  return [
    {
      id: "override",
      label: "Edit key name...",
      icon: <PenWriting3 iconSize="md-medium" />,
      ActionComponent: (props) => <EditKeyName {...props} keyDetails={key} />,
    },
    {
      id: "copy",
      label: "Copy key ID",
      icon: <Clone iconSize="md-medium" />,
      onClick: () => {
        navigator.clipboard
          .writeText(key.id)
          .then(() => {
            toast.success("Key ID copied to clipboard");
          })
          .catch((error) => {
            console.error("Failed to copy to clipboard:", error);
            toast.error("Failed to copy to clipboard");
          });
      },
      divider: true,
    },
    {
      id: "edit-external-id",
      label: "Edit External ID...",
      icon: <ArrowOppositeDirectionY iconSize="md-medium" />,
      ActionComponent: (props) => <EditExternalId {...props} keyDetails={key} />,
    },
    ...(key.identity?.external_id
      ? [
          {
            id: "copy-external-id",
            label: "Copy External ID",
            icon: <Clone iconSize="md-medium" />,
            onClick: () => {
              navigator.clipboard
                // Empty case cannot happen since this will only render if identity exists
                .writeText(key.identity?.external_id ?? "")
                .then(() => toast.success("External ID copied to clipboard"))
                .catch((error) => {
                  console.error("Failed to copy to clipboard:", error);
                  toast.error("Failed to copy to clipboard");
                });
            },
            divider: true,
          },
        ]
      : []),
    {
      id: key.enabled ? "disable-key" : "enable-key",
      label: key.enabled ? "Disable Key..." : "Enable Key...",
      icon: key.enabled ? <Ban iconSize="md-medium" /> : <Check iconSize="md-medium" />,
      ActionComponent: (props) => <UpdateKeyStatus {...props} keyDetails={key} />,
      divider: true,
    },
    {
      id: "edit-credits",
      label: "Edit credits...",
      icon: <ChartPie iconSize="md-medium" />,
      ActionComponent: (props) => <EditCredits {...props} keyDetails={key} />,
    },
    {
      id: "edit-ratelimit",
      label: "Edit ratelimit...",
      icon: <Gauge iconSize="md-medium" />,
      ActionComponent: (props) => <EditRatelimits {...props} keyDetails={key} />,
    },
    {
      id: "edit-expiration",
      label: "Edit expiration...",
      icon: <CalendarClock iconSize="md-medium" />,
      ActionComponent: (props) => <EditExpiration {...props} keyDetails={key} />,
    },
    {
      id: "edit-metadata",
      label: "Edit metadata...",
      icon: <Code iconSize="md-medium" />,
      ActionComponent: (props) => <EditMetadata {...props} keyDetails={key} />,
      divider: true,
    },
    {
      id: "edit-rbac",
      label: "Manage roles and permissions...",
      icon: <Tag iconSize="md-medium" />,
      ActionComponent: (props) => (
        <KeyRbacDialog
          {...props}
          existingKey={{
            id: key.id,
            permissionIds: [],
            roleIds: [],
            name: key.name ?? undefined,
          }}
        />
      ),
      prefetch: async () => {
        try {
          // Primary data - always needed when dialog opens
          const connectedData = await trpcUtils.key.connectedRolesAndPerms.fetch({
            keyId: key.id,
          });

          const currentRoleIds = connectedData?.roles?.map((r) => r.id) ?? [];
          const directPermissionIds =
            connectedData?.permissions?.filter((p) => p.source === "direct")?.map((p) => p.id) ??
            [];
          const rolePermissionIds =
            connectedData?.permissions?.filter((p) => p.source === "role")?.map((p) => p.id) ?? [];
          const allEffectivePermissionIds = [...rolePermissionIds, ...directPermissionIds];

          // Prefetch dependent data that requires connectedData
          const dependentPrefetches = [];

          if (allEffectivePermissionIds.length > 0 || currentRoleIds.length > 0) {
            dependentPrefetches.push(
              trpcUtils.key.queryPermissionSlugs.prefetch({
                roleIds: currentRoleIds,
                permissionIds: allEffectivePermissionIds,
              }),
            );
          }

          // Always prefetch combobox data - independent of connectedData
          const comboboxDataPromise = Promise.all([
            trpcUtils.key.update.rbac.permissions.query.prefetchInfinite({
              limit: MAX_PERMS_FETCH_LIMIT,
            }),
            trpcUtils.key.update.rbac.roles.query.prefetchInfinite({
              limit: MAX_ROLES_FETCH_LIMIT,
            }),
            trpcUtils.authorization.roles.keys.query.prefetchInfinite({
              limit: MAX_KEYS_FETCH_LIMIT,
            }),
            trpcUtils.authorization.roles.permissions.query.prefetchInfinite({
              limit: MAX_PERMS_FETCH_LIMIT,
            }),
          ]);

          await Promise.all([comboboxDataPromise, ...dependentPrefetches]);
        } catch {
          // Fallback: prefetch only the combobox data which doesn't depend on connectedData
          try {
            await Promise.all([
              trpcUtils.key.update.rbac.permissions.query.prefetchInfinite({
                limit: MAX_PERMS_FETCH_LIMIT,
              }),
              trpcUtils.key.update.rbac.roles.query.prefetchInfinite({
                limit: MAX_ROLES_FETCH_LIMIT,
              }),
              trpcUtils.authorization.roles.keys.query.prefetchInfinite({
                limit: MAX_KEYS_FETCH_LIMIT,
              }),
              trpcUtils.authorization.roles.permissions.query.prefetchInfinite({
                limit: MAX_PERMS_FETCH_LIMIT,
              }),
            ]);
          } catch (fallbackError) {
            console.warn("Failed to prefetch combobox data:", fallbackError);
          }
        }
      },
      divider: true,
    },
    ...(apiId
      ? [
          {
            id: "rotate-key",
            label: "Rotate key...",
            icon: <ArrowDottedRotateAnticlockwise iconSize="md-medium" />,
            ActionComponent: (props) => (
              <RotateKey {...props} keyDetails={key} apiId={apiId} keyspaceId={keyspaceId} />
            ),
            divider: true,
          } satisfies MenuItem,
        ]
      : []),
    {
      id: "delete-key",
      label: "Delete key",
      icon: <Trash iconSize="md-medium" />,
      ActionComponent: (props) => <DeleteKey {...props} keyDetails={key} />,
    },
  ];
};

type KeysTableActionsProps = {
  keyData: KeyDetails;
  apiId?: string;
  keyspaceId?: string | null;
};

export const KeysTableActions = ({ keyData, apiId, keyspaceId }: KeysTableActionsProps) => {
  const trpcUtils = trpc.useUtils();
  const items = getKeysTableActionItems(keyData, trpcUtils, { apiId, keyspaceId });
  return <TableActionPopover items={items} />;
};
