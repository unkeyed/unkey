import { RotateKeyDialog } from "@/components/api-keys-table/components/actions/components/rotate-key/rotate-key-dialog";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { trpc } from "@/lib/trpc/client";
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
      keyName={rootKeyDetails.name}
      mutation={mutation}
      resourceLabel="root key"
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
