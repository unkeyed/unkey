import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";

type CreateProjectResponse = {
  id: string;
  name: string;
  slug: string;
  gitRepositoryUrl?: string | null;
  createdAt: number;
};

export const useCreateProject = (onSuccess: (data: CreateProjectResponse) => void) => {
  const project = trpc.project.create.useMutation({
    onSuccess(data) {
      onSuccess(data);
    },
    onError(err) {
      if (err.data?.code === "NOT_FOUND") {
        toast.error("Project Creation Failed", {
          description: "Unable to find the workspace. Please refresh and try again.",
        });
      } else if (err.data?.code === "CONFLICT") {
        toast.error("Project Already Exists", {
          description: err.message || "A project with this slug already exists in your workspace.",
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We encountered an issue while creating your project. Please try again later or contact support at support@unkey.dev",
        });
      } else if (err.data?.code === "BAD_REQUEST") {
        toast.error("Invalid Configuration", {
          description: `Please check your project settings. ${err.message || ""}`,
        });
      } else if (err.data?.code === "FORBIDDEN") {
        toast.error("Permission Denied", {
          description:
            err.message || "You don't have permission to create projects in this workspace.",
        });
      } else {
        toast.error("Failed to Create Project", {
          description: err.message || "An unexpected error occurred. Please try again later.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("https://support.unkey.dev", "_blank"),
          },
        });
      }
    },
  });
  return project;
};
