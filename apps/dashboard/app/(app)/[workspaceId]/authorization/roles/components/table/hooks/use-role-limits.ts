import { trpc } from "@/lib/trpc/client";

export type RoleLimitState = {
  totalKeys: number;
  totalPerms: number;
  hasKeyWarning: boolean;
  hasPermWarning: boolean;
  shouldPrefetch: boolean;
  shouldAllowEdit: boolean;
};

// `MAX_ATTACH_LIMIT` threshold for role attachments. Beyond this limit:
// - Role editing is disabled to prevent UI performance degradation
// - Warning callouts are shown to inform users of potential slowdowns
// - Prefetching of connected keys/permissions is skipped to reduce API load
export const MAX_ATTACH_LIMIT = 50;

export const useRoleLimits = (roleId?: string) => {
  const trpcUtils = trpc.useUtils();

  const getKeysPreview = () => {
    if (!roleId) {
      return null;
    }
    return trpcUtils.authorization.roles.connectedKeys.getData({
      roleId,
    });
  };

  const getPermsPreview = () => {
    if (!roleId) {
      return null;
    }
    return trpcUtils.authorization.roles.connectedPerms.getData({
      roleId,
    });
  };

  const calculateLimits = (
    additionalKeys?: string[],
    additionalPerms?: string[],
  ): RoleLimitState => {
    const keysPreview = getKeysPreview();
    const permsPreview = getPermsPreview();

    // Calculate totals - use preview data first, fallback to additional arrays
    const totalKeys = keysPreview?.totalCount || additionalKeys?.length || 0;

    const totalPerms = permsPreview?.totalCount || additionalPerms?.length || 0;

    // Only show warnings for existing roles (edit mode)
    const hasKeyWarning = Boolean(roleId && totalKeys > MAX_ATTACH_LIMIT);
    const hasPermWarning = Boolean(roleId && totalPerms > MAX_ATTACH_LIMIT);

    // Should prefetch when both are under limit
    const shouldPrefetch = totalKeys <= MAX_ATTACH_LIMIT && totalPerms <= MAX_ATTACH_LIMIT;

    // Should allow editing when both are under limit (or it's create mode)
    const shouldAllowEdit = !roleId || shouldPrefetch;

    return {
      totalKeys,
      totalPerms,
      hasKeyWarning,
      hasPermWarning,
      shouldPrefetch,
      shouldAllowEdit,
    };
  };

  const prefetchIfAllowed = async () => {
    if (!roleId) {
      return;
    }

    const { shouldPrefetch } = calculateLimits();

    if (shouldPrefetch) {
      await trpcUtils.authorization.roles.connectedKeysAndPerms.prefetch({
        roleId,
      });
    }
  };

  return {
    calculateLimits,
    prefetchIfAllowed,
    MAX_ATTACH_LIMIT,
  };
};
