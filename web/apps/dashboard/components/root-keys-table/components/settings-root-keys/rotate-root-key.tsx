import { RotateKeyDialog } from "@/components/api-keys-table/components/actions/components/rotate-key/rotate-key-dialog";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { useRotateRootKey } from "../../hooks/use-rotate-root-key";
import { RootKeyInfo } from "./root-key-info";

type RotateRootKeyProps = { rootKeyDetails: RootKey } & ActionComponentProps;

export const RotateRootKey = ({ rootKeyDetails, isOpen, onClose }: RotateRootKeyProps) => {
  const workspace = useWorkspaceNavigation();
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
      detailsHref={(rotated) => `/${workspace.slug}/settings/root-keys/${rotated.id}`}
      onRotated={() => {
        trpcUtils.settings.rootKeys.query.invalidate();
      }}
    />
  );
};
