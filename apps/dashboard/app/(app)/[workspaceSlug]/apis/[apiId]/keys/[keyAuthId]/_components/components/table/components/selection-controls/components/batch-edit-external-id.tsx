import { ExternalIdField } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/components/external-id-field";
import { ConfirmPopover } from "@/components/confirmation-popover";
import { TriangleWarning2 } from "@unkey/icons";
import { Button, DialogContainer } from "@unkey/ui";
import { useRef, useState } from "react";
import { useBatchEditExternalId } from "../../actions/components/hooks/use-edit-external-id";

type BatchEditExternalIdProps = {
  selectedKeyIds: string[];
  keysWithExternalIds: number; // Count of keys that already have external IDs
  isOpen: boolean;
  onClose: () => void;
};

export const BatchEditExternalId = ({
  selectedKeyIds,
  keysWithExternalIds,
  isOpen,
  onClose,
}: BatchEditExternalIdProps): JSX.Element => {
  const [selectedIdentityId, setSelectedIdentityId] = useState<string | null>(null);
  const [selectedExternalId, setSelectedExternalId] = useState<string | null>(null);
  const [isConfirmPopoverOpen, setIsConfirmPopoverOpen] = useState(false);
  const clearButtonRef = useRef<HTMLButtonElement>(null);

  const updateKeyOwner = useBatchEditExternalId(() => {
    onClose();
  });

  const handleSubmit = () => {
    updateKeyOwner.mutate({
      keyIds: selectedKeyIds,
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
    await updateKeyOwner.mutateAsync({
      keyIds: selectedKeyIds,
      ownerType: "v2",
      identity: {
        id: null,
        externalId: null,
      },
    });
  };

  const totalKeys = selectedKeyIds.length;
  const hasKeysWithExternalIds = keysWithExternalIds > 0;

  // Determine what button to show based on whether a new external ID is selected
  const showUpdateButton = selectedIdentityId !== null;

  return (
    <>
      <DialogContainer
        isOpen={isOpen}
        subTitle={`Provide an External ID to ${totalKeys} selected ${
          totalKeys === 1 ? "key" : "keys"
        }, like a userID from your system`}
        onOpenChange={handleDialogOpenChange}
        title="Edit External IDs"
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <div className="w-full flex gap-2">
              {showUpdateButton ? (
                <Button
                  type="button"
                  variant="primary"
                  size="xlg"
                  className="rounded-lg flex-1"
                  loading={updateKeyOwner.isLoading}
                  onClick={handleSubmit}
                >
                  Update External ID
                </Button>
              ) : (
                <Button
                  type="button"
                  variant="primary"
                  color="danger"
                  size="xlg"
                  className="rounded-lg flex-1"
                  loading={updateKeyOwner.isLoading}
                  onClick={handleClearButtonClick}
                  ref={clearButtonRef}
                  disabled={!hasKeysWithExternalIds}
                >
                  Clear External ID
                </Button>
              )}
            </div>
            {hasKeysWithExternalIds && (
              <div className="text-gray-9 text-xs mt-2">
                Note: {keysWithExternalIds} out of {totalKeys} selected{" "}
                {totalKeys === 1 ? "key" : "keys"} already{" "}
                {keysWithExternalIds === 1 ? "has" : "have"} an External ID
              </div>
            )}
            <div className="text-gray-9 text-xs">Changes will be applied immediately</div>
          </div>
        }
      >
        {hasKeysWithExternalIds && (
          <div className="rounded-xl bg-errorA-2 dark:bg-black border border-errorA-3 flex items-center gap-4 px-[22px] py-6 mb-4">
            <div className="bg-error-9 size-8 rounded-full flex items-center justify-center flex-shrink-0">
              <TriangleWarning2 iconSize="sm-regular" className="text-white" />
            </div>
            <div className="text-error-12 text-[13px] leading-6">
              <span className="font-medium">Warning:</span>{" "}
              {keysWithExternalIds === totalKeys ? (
                <>
                  All selected keys already have External IDs. Setting a new ID will override the
                  existing ones.
                </>
              ) : (
                <>
                  Some selected keys already have External IDs. Setting a new ID will override the
                  existing ones.
                </>
              )}
            </div>
          </div>
        )}
        <div className="my-2">
          <ExternalIdField
            value={selectedIdentityId}
            onChange={(identityId: string | null, externalId: string | null) => {
              setSelectedIdentityId(identityId);
              setSelectedExternalId(externalId);
            }}
            disabled={updateKeyOwner.isLoading}
          />
        </div>
      </DialogContainer>
      <ConfirmPopover
        isOpen={isConfirmPopoverOpen}
        onOpenChange={setIsConfirmPopoverOpen}
        onConfirm={clearSelection}
        triggerRef={clearButtonRef}
        title={`Confirm removing External ${keysWithExternalIds > 1 ? "IDs" : "ID"}`}
        description={`This will remove the External ID association from ${keysWithExternalIds} ${
          keysWithExternalIds === 1 ? "key" : "keys"
        }. Any tracking or analytics related to ${
          keysWithExternalIds === 1 ? "this ID" : "these IDs"
        } will no longer be associated with ${
          keysWithExternalIds === 1 ? "this key" : "these keys"
        }.`}
        confirmButtonText={`Remove External ${keysWithExternalIds > 1 ? "IDs" : "ID"}`}
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
};
