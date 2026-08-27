import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { trpc } from "@/lib/trpc/client";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { useRotateKey } from "./hooks/use-rotate-key";
import { RotateKeyDialog } from "./rotate-key-dialog";

type RotateKeyProps = {
  keyDetails: KeyDetails;
} & ActionComponentProps;

export function RotateKey({ keyDetails, isOpen, onClose }: RotateKeyProps) {
  const trpcUtils = trpc.useUtils();
  const mutation = useRotateKey();

  return (
    <RotateKeyDialog
      keyId={keyDetails.id}
      keyName={keyDetails.name}
      mutation={mutation}
      resourceLabel="key"
      isOpen={isOpen}
      onClose={onClose}
      onRotated={() => {
        trpcUtils.api.keys.list.invalidate();
      }}
    />
  );
}
