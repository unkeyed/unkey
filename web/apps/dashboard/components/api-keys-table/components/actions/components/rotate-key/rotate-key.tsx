import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { trpc } from "@/lib/trpc/client";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { KeyInfo } from "../key-info";
import { useRotateKey } from "./hooks/use-rotate-key";
import { RotateKeyDialog } from "./rotate-key-dialog";

type RotateKeyProps = {
  keyDetails: KeyDetails;
  apiId: string;
  keyspaceId?: string | null;
} & ActionComponentProps;

export const RotateKey = ({ keyDetails, apiId, keyspaceId, isOpen, onClose }: RotateKeyProps) => {
  const trpcUtils = trpc.useUtils();
  const mutation = useRotateKey();

  return (
    <RotateKeyDialog
      keyId={keyDetails.id}
      info={<KeyInfo keyDetails={keyDetails} />}
      mutation={mutation}
      apiId={apiId}
      keyspaceId={keyspaceId}
      resourceLabel="key"
      formId="rotate-key-form"
      isOpen={isOpen}
      onClose={onClose}
      onRotated={() => {
        trpcUtils.api.keys.list.invalidate();
        trpcUtils.api.overview.keyCount.invalidate();
      }}
    />
  );
};
