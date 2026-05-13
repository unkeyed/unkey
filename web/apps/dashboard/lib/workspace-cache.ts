/**
 * Cache invalidation helper for when workspace data changes
 * Can invalidate specific org's workspace or all workspaces
 */
export const invalidateWorkspaceCache = async (orgId?: string) => {
  const { revalidateTag } = await import("next/cache");

  if (orgId) {
    // Invalidate specific org's workspace cache
    revalidateTag(`workspace-${orgId}`, "max");
  } else {
    // Invalidate all workspace caches
    revalidateTag("workspace", "max");
  }
};
