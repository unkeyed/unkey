"use client";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";

export const useFetchPermissionSlugs = (
  roleIds: string[] = [],
  permissionIds: string[] = [],
  enabled = true,
) => {
  const { data, isLoading, error, refetch } = trpc.key.queryPermissionSlugs.useQuery(
    {
      roleIds,
      permissionIds,
    },
    {
      enabled,
      onError(err) {
        if (err.data?.code === "BAD_REQUEST") {
          toast.error("Invalid Roles or Permissions", {
            description:
              "One or more selected roles or permissions could not be found. Please refresh and try again.",
          });
        } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
          toast.error("Server Error", {
            description:
              "We were unable to resolve permission slugs. Please try again or contact support@unkey.dev",
          });
        } else {
          toast.error("Failed to Resolve Permission Slugs", {
            description:
              err.message ||
              "An unexpected error occurred. Please try again or contact support@unkey.dev",
            action: {
              label: "Contact Support",
              onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
            },
          });
        }
      },
    },
  );

  return {
    data,
    isLoading,
    error,
    refetch,
    hasData: !isLoading && data !== undefined,
  };
};
