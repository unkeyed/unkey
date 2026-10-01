import { RotateKeyDialog } from "@/components/api-keys-table/components/actions/components/rotate-key/rotate-key-dialog";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { rerollRootKey, rootKeysV2QueryKeys } from "@/lib/root-keys-api";
import { trpc } from "@/lib/trpc/client";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { RootKeyNameCell, toast } from "@unkey/ui";
import { useRotateRootKey } from "./hooks/use-rotate-root-key";

type RotateRootKeyProps = {
  rootKeyDetails: { id: string; name: string | null };
  onRotated?: () => void;
} & ActionComponentProps;

export function RotateRootKey({ rootKeyDetails, isOpen, onClose, onRotated }: RotateRootKeyProps) {
  const trpcUtils = trpc.useUtils();
  const mutation = useRotateRootKey();

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
        trpcUtils.settings.rootKeys.query.invalidate();
        trpcUtils.settings.rootKeys.get.invalidate({ keyId: rootKeyDetails.id });
        onRotated?.();
      }}
    />
  );
}

export function RotateRootKeyV2({
  rootKeyDetails,
  isOpen,
  onClose,
  onRotated,
}: RotateRootKeyProps) {
  const workspace = useWorkspaceNavigation();
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: rerollRootKey,
    onError: (error) => {
      toast.error("Failed to rotate Root Key", {
        description: error instanceof Error ? error.message : "Please try again later.",
      });
    },
  });

  return (
    <RotateKeyDialog
      keyId={rootKeyDetails.id}
      info={<RootKeyNameCell name={rootKeyDetails.name ?? undefined} />}
      mutation={mutation}
      resourceLabel="root key"
      formId="rotate-root-key-v2-form"
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
