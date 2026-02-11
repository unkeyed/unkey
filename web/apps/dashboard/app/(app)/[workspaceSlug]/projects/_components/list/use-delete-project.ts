import { collectionManager } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

export const useDeleteProject = (onSuccess: () => void) => {
  const trpcUtils = trpc.useUtils();

  return trpc.deploy.project.delete.useMutation({
    async onSuccess(_data, variables) {
      await Promise.all([
        trpcUtils.deploy.project.list.invalidate(),
        collectionManager.cleanup(variables.projectId),
      ]);

      toast.success("Project deleted", {
        description: "The project has been permanently removed.",
      });

      onSuccess();
    },
    onError(err) {
      switch (err.data?.code) {
        case "NOT_FOUND":
          toast.error("Project not found", {
            description: "This project no longer exists or you don't have access to it.",
          });
          return;
        case "PRECONDITION_FAILED":
          toast.error("Delete protection enabled", {
            description: err.message,
          });
          return;
        case "TOO_MANY_REQUESTS":
          toast.error("Too many requests", {
            description: "Please wait a moment and try again.",
          });
          return;
        default:
          toast.error("Failed to delete project", {
            description: err.message || "An unexpected error occurred. Please try again later.",
            action: {
              label: "Contact Support",
              onClick: () => window.open("mailto:support@unkey.com", "_blank"),
            },
          });
      }
    },
  });
};

