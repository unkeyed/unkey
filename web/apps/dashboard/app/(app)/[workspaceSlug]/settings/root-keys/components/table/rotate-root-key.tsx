import { RotateKeyDialog } from "@/components/api-keys-table/components/actions/components/rotate-key/rotate-key-dialog";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { trpc } from "@/lib/trpc/client";
import { useRotateRootKey } from "./hooks/use-rotate-root-key";
import { RootKeyInfo, type RootKeySummary } from "./root-key-info";

type RotateRootKeyProps = { rootKeyDetails: RootKeySummary } & ActionComponentProps;

export const RotateRootKey = ({ rootKeyDetails, isOpen, onClose }: RotateRootKeyProps) => {
  const trpcUtils = trpc.useUtils();
  const mutation = useRotateRootKey();

  return (
    <RotateKeyDialog
      keyId={rootKeyDetails.id}
      info={<RootKeyInfo rootKeyDetails={rootKeyDetails} />}
      mutation={mutation}
      resourceLabel="root key"
      formId="rotate-root-key-form"
      isOpen={isOpen}
      onClose={onClose}
      onRotated={() => {
        trpcUtils.settings.rootKeys.query.invalidate();
        trpcUtils.settings.rootKeys.get.invalidate({ keyId: rootKeyDetails.id });
      }}
    />
  );
};
