import { RotateKeyDialog } from "@/components/api-keys-table/components/actions/components/rotate-key/rotate-key-dialog";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { ROOT_KEYS_V2_QUERY_KEY } from "@/lib/root-keys-api";
import { trpc } from "@/lib/trpc/client";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { useQueryClient } from "@tanstack/react-query";
import { useRotateRootKey } from "../../hooks/use-rotate-root-key";
import { useRotateRootKeyV2 } from "../../hooks/use-rotate-root-key-v2";
import { useRootKeysTransport } from "../../root-keys-transport";
import { RootKeyInfo } from "./root-key-info";

type RotateRootKeyProps = { rootKeyDetails: RootKey } & ActionComponentProps;

export const RotateRootKey = (props: RotateRootKeyProps) => {
  const transport = useRootKeysTransport();
  return transport === "v2" ? <V2RotateRootKey {...props} /> : <LegacyRotateRootKey {...props} />;
};

const LegacyRotateRootKey = ({ rootKeyDetails, isOpen, onClose }: RotateRootKeyProps) => {
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
      }}
    />
  );
};

const V2RotateRootKey = ({ rootKeyDetails, isOpen, onClose }: RotateRootKeyProps) => {
  const queryClient = useQueryClient();
  const mutation = useRotateRootKeyV2();

  return (
    <RotateKeyDialog
      keyId={rootKeyDetails.id}
      info={<RootKeyInfo rootKeyDetails={rootKeyDetails} />}
      mutation={mutation}
      resourceLabel="root key"
      formId="rotate-root-key-form"
      isOpen={isOpen}
      onClose={onClose}
      onRotated={() => queryClient.invalidateQueries({ queryKey: ROOT_KEYS_V2_QUERY_KEY })}
    />
  );
};
