import { ExternalIdField } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/components/external-id-field";
import { ConfirmPopover } from "@/components/confirmation-popover";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { Button, DialogContainer } from "@unkey/ui";
import { useRef, useState } from "react";
import { useEditExternalId } from "../hooks/use-edit-external-id";
import { KeyInfo } from "../key-info";

type EditExternalIdProps = {
  keyDetails: KeyDetails;
} & ActionComponentProps;

export const EditExternalId = ({
  keyDetails,
  isOpen,
  onClose,
}: EditExternalIdProps): JSX.Element => {
  const [originalIdentityId, setOriginalIdentityId] = useState<string | null>(
    keyDetails.identity_id || null,
  );
  const [selectedIdentityId, setSelectedIdentityId] = useState<string | null>(
    keyDetails.identity_id || null,
  );
  const [selectedExternalId, setSelectedExternalId] = useState<string | null>(null);
  const [isConfirmPopoverOpen, setIsConfirmPopoverOpen] = useState(false);
  const clearButtonRef = useRef<HTMLButtonElement>(null);

  const updateKeyOwner = useEditExternalId(() => {
    setOriginalIdentityId(selectedIdentityId);
    onClose();
  });

  const handleSubmit = () => {
    updateKeyOwner.mutate({
      keyIds: keyDetails.id,
      ownerType: "v2",
      identity: {
        id: selectedIdentityId,
        externalId: selectedExternalId,
      },
    });
  };

  const handleClearButtonClick = () => {
    setIsConfirmPopoverOpen(true);
  };

  const handleDialogOpenChange = (open: boolean) => {
    if (isConfirmPopoverOpen && !isOpen) {
      // If confirm popover is active don't let this trigger outer popover
      return;
    }

    if (!isConfirmPopoverOpen && !open) {
      onClose();
    }
  };

  const clearSelection = async () => {
    setSelectedIdentityId(null);
    await updateKeyOwner.mutateAsync({
      keyIds: keyDetails.id,
      ownerType: "v2",
      identity: {
        id: null,
      },
    });
  };

  return (
    <>
      <DialogContainer
        isOpen={isOpen}
        subTitle="Provide an owner to this key, like a userID from your system"
        onOpenChange={handleDialogOpenChange}
        title="Edit External ID"
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <div className="w-full flex gap-2">
              {originalIdentityId !== null ? (
                <Button
                  type="button"
                  variant="primary"
                  color="danger"
                  size="xlg"
                  className="rounded-lg flex-1"
                  loading={updateKeyOwner.isPending}
                  onClick={handleClearButtonClick}
                  ref={clearButtonRef}
                >
                  Clear External ID
                </Button>
              ) : (
                <Button
                  type="button"
                  form="edit-external-id-form"
                  variant="primary"
                  size="xlg"
                  className="rounded-lg flex-1"
                  loading={updateKeyOwner.isPending}
                  onClick={handleSubmit}
                  disabled={!originalIdentityId && !selectedIdentityId}
                >
                  Update External ID
                </Button>
              )}
            </div>
            <div className="text-gray-9 text-xs">Changes will be applied immediately</div>
          </div>
        }
      >
        <KeyInfo keyDetails={keyDetails} />
        <div className="py-1 my-2">
          <div className="h-[1px] bg-grayA-3 w-full" />
        </div>
        <ExternalIdField
          value={selectedIdentityId}
          onChange={(identityId: string | null, externalId: string | null) => {
            setSelectedIdentityId(identityId);
            setSelectedExternalId(externalId);
          }}
          currentIdentity={
            keyDetails.identity_id
              ? {
                  id: keyDetails.identity_id,
                  externalId: keyDetails.owner_id || "",
                }
              : undefined
          }
          disabled={updateKeyOwner.isPending || Boolean(originalIdentityId)}
        />
      </DialogContainer>
      <ConfirmPopover
        isOpen={isConfirmPopoverOpen}
        onOpenChange={setIsConfirmPopoverOpen}
        onConfirm={clearSelection}
        triggerRef={clearButtonRef}
        title="Confirm removing External ID"
        description="This will remove the External ID association from this key. Any tracking or analytics related to this ID will no longer be associated with this key."
        confirmButtonText="Remove External ID"
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
};
