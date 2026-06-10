import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";
import { resourceUrl } from "./resource-url";

// Shape returned by trpc.settings.scheduledDeletions.list. Stays a
// discriminated union over resourceType so adding a new resource on
// the backend automatically flows through here.
const schema = z.object({
  resourceType: z.enum(["project"]),
  resourceId: z.string(),
  name: z.string(),
  deletePermanentlyAt: z.number().int(),
});

export type ScheduledDeletion = z.infer<typeof schema>;

const key = (row: ScheduledDeletion) => `${row.resourceType}:${row.resourceId}`;

// scheduledDeletions tracks every workspace resource currently in its
// hard-delete grace window. Deleting from the collection means
// "restore" — that's the only client-driven mutation that removes a
// row from this view (the alternative is the cron sweep elapsing the
// grace window, which removes it server-side and a refetch picks up).
export const scheduledDeletions = createCollection<ScheduledDeletion, string>(
  queryCollectionOptions({
    queryClient,
    queryKey: ["scheduled-deletions"],
    retry: 3,
    refetchInterval: 5000,
    queryFn: async () => {
      return await trpcClient.settings.scheduledDeletions.list.query();
    },
    getKey: key,
    onDelete: async ({ transaction }) => {
      const mutation = transaction.mutations[0];
      const row = mutation.original;

      const loadingId = toast.loading(`Restoring ${row.name}...`);

      try {
        await trpcClient.settings.scheduledDeletions.restore.mutate({
          resourceType: row.resourceType,
          resourceId: row.resourceId,
        });
      } catch (err) {
        toast.dismiss(loadingId);
        console.error("Failed to restore", err);
        const code = (err as { data?: { code?: string } })?.data?.code;
        const message = (err as { message?: string })?.message;
        switch (code) {
          case "NOT_FOUND":
            toast.error("Restore failed", {
              description:
                "The resource is no longer scheduled for deletion. It may have been permanently removed already.",
            });
            break;
          case "FORBIDDEN":
            toast.error("Permission denied", {
              description: "You don't have permission to restore this resource.",
            });
            break;
          default:
            toast.error("Failed to restore", {
              description: message || "An unexpected error occurred. Please try again.",
            });
        }
        throw err;
      }

      // resourceUrl reads window.location at click time; if we can't
      // resolve it (server, unexpected route) we drop the Open action
      // rather than render a broken link.
      const url = resourceUrl(row.resourceType, row.resourceId);

      toast.success(`Restored ${row.name}`, {
        id: loadingId,
        duration: 8000,
        ...(url
          ? {
              action: {
                label: "Open",
                onClick: () => {
                  window.location.href = url;
                },
              },
            }
          : {}),
      });
    },
  }),
);
