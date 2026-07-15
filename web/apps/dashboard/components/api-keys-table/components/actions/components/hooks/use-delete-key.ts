import { trpc } from "@/lib/trpc/client";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import type { Unkey } from "@unkey/api";
import {
  BadRequestErrorResponse,
  ForbiddenErrorResponse,
  InternalServerErrorResponse,
  NotFoundErrorResponse,
  UnauthorizedErrorResponse,
} from "@unkey/api/models/errors";
import { toast } from "@unkey/ui";

type DeleteKeyRequest = Parameters<Unkey["keys"]["deleteKey"]>[0];

type DeleteKeyVariables = {
  keyIds: DeleteKeyRequest["keyId"][];
};

type DeleteKeyResult = {
  totalDeleted: number;
};

export const useDeleteKey = (onSuccess?: () => void) => {
  const trpcUtils = trpc.useUtils();
  const deleteKey = useMutation<DeleteKeyResult, unknown, DeleteKeyVariables>({
    mutationFn: async ({ keyIds }) => {
      if (keyIds.length === 0) {
        throw new Error("No keys were provided for deletion");
      }

      const deletedKeyIds = await Promise.all(
        keyIds.map(async (keyId) => {
          await getUnkeyClient().keys.deleteKey({ keyId });
          return keyId;
        }),
      );

      return {
        totalDeleted: deletedKeyIds.length,
      };
    },
    onSuccess(data) {
      const deletedCount = data.totalDeleted;

      if (deletedCount === 1) {
        toast.success("Key Deleted", {
          description: "Your key has been permanently deleted successfully",
          duration: 5000,
        });
      } else {
        toast.success("Keys Deleted", {
          description: `${deletedCount} keys have been permanently deleted successfully`,
          duration: 5000,
        });
      }

      trpcUtils.api.keys.list.invalidate();
      if (onSuccess) {
        onSuccess();
      }
    },
    onError(err, variable) {
      const errorMessage = deleteKeyErrorMessage(err);
      const errorCode = deleteKeyErrorCode(err);
      const isPlural = variable.keyIds.length > 1;
      const keyText = isPlural ? "keys" : "key";

      if (errorCode === "NOT_FOUND") {
        toast.error("Key Deletion Failed", {
          description: `Unable to find the ${keyText}. Please refresh and try again.`,
        });
      } else if (errorCode === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description: `We encountered an issue while deleting your ${keyText}. Please try again later or contact support at support.unkey.dev`,
        });
      } else if (errorCode === "FORBIDDEN") {
        toast.error("Permission Denied", {
          description: `You don't have permission to delete ${
            isPlural ? "these keys" : "this key"
          }.`,
        });
      } else {
        toast.error(`Failed to Delete ${isPlural ? "Keys" : "Key"}`, {
          description: errorMessage || "An unexpected error occurred. Please try again later.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.com", "_blank"),
          },
        });
      }
    },
  });

  return deleteKey;
};

function deleteKeyErrorCode(
  error: unknown,
): "BAD_REQUEST" | "FORBIDDEN" | "INTERNAL_SERVER_ERROR" | "NOT_FOUND" | "UNKNOWN" {
  if (error instanceof NotFoundErrorResponse) {
    return "NOT_FOUND";
  }
  if (error instanceof ForbiddenErrorResponse || error instanceof UnauthorizedErrorResponse) {
    return "FORBIDDEN";
  }
  if (error instanceof BadRequestErrorResponse) {
    return "BAD_REQUEST";
  }
  if (error instanceof InternalServerErrorResponse) {
    return "INTERNAL_SERVER_ERROR";
  }

  return "UNKNOWN";
}

function deleteKeyErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return "An unexpected error occurred. Please try again later.";
}
