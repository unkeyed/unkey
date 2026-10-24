import { RotateKeyDialog } from "@/components/api-keys-table/components/actions/components/rotate-key/rotate-key-dialog";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { rootKeysV2QueryKeys } from "@/lib/root-keys-api";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { V2RootKeysRerollKeyRequestBody } from "@unkey/api/models/components";
import { RootKeyNameCell, toast } from "@unkey/ui";

type RotateRootKeyProps = {
  rootKeyDetails: { id: string; name: string | null };
  onRotated?: () => void;
} & ActionComponentProps;

export function RotateRootKey({ rootKeyDetails, isOpen, onClose, onRotated }: RotateRootKeyProps) {
  const workspace = useWorkspaceNavigation();
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: async (request: V2RootKeysRerollKeyRequestBody) => {
      const response = await getUnkeyClient().rootKeys.rerollKey(request);
      return response.data;
    },
    onError: (error) => {
      toast.error("Failed to rotate Root Key", {
        description: getErrorMessage(error, "Please try again later."),
      });
    },
  });

  return (
    <RotateKeyDialog
      keyId={rootKeyDetails.id}
      info={<RootKeyNameCell name={rootKeyDetails.name ?? undefined} />}
      mutation={mutation}
      resourceLabel="root key"
      formId="rotate-root-key-form"
      isOpen={isOpen}
      onClose={onClose}
      onRotated={() => {
        void queryClient.invalidateQueries({
          queryKey: rootKeysV2QueryKeys.workspace(workspace.id),
        });
        onRotated?.();
      }}
    />
  );
}
