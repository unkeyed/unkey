import { useFetchIdentities } from "@/app/(app)/apis/[apiId]/_components/create-key/hooks/use-fetch-identities";
import { createIdentityOptions } from "@/app/(app)/apis/[apiId]/_components/create-key/hooks/use-fetch-identities/create-identity-options";
import { ConfirmPopover } from "@/components/confirmation-popover";
import { DialogContainer } from "@/components/dialog-container";
import { FormCombobox } from "@/components/ui/form-combobox";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useRef, useState } from "react";
import type { ActionComponentProps } from "../../keys-table-action.popover";
import { useCreateIdentity } from "../hooks/use-create-identity";
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
  const [isConfirmPopoverOpen, setIsConfirmPopoverOpen] = useState(false);
  const [searchValue, setSearchValue] = useState("");
  const clearButtonRef = useRef<HTMLButtonElement>(null);

  const { identities, isFetchingNextPage, hasNextPage, loadMore } = useFetchIdentities();
  const identityOptions = createIdentityOptions({
    identities,
    hasNextPage,
    isFetchingNextPage,
    loadMore,
  });

  const updateKeyOwner = useEditExternalId(() => {
    setOriginalIdentityId(selectedIdentityId);
    onClose();
  });

  // Add the create identity hook
  const createIdentity = useCreateIdentity((data) => {
    // When identity is created, select it
    setSelectedIdentityId(data.identityId);
  });

  const handleSubmit = () => {
    updateKeyOwner.mutate({
      keyId: keyDetails.id,
      ownerType: "v2",
      identity: {
        id: selectedIdentityId,
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
      keyId: keyDetails.id,
      ownerType: "v2",
      identity: {
        id: null,
      },
    });
  };

  const handleCreateIdentity = () => {
    if (searchValue.trim()) {
      createIdentity.mutate({
        externalId: searchValue.trim(),
        meta: null,
      });
    }
  };

  return (
    <>
      <DialogContainer
        isOpen={isOpen}
        subTitle="Provide an owner to this key, like a userId from your system"
        onOpenChange={handleDialogOpenChange}
        title="Edit external ID"
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
                  loading={updateKeyOwner.isLoading}
                  onClick={handleClearButtonClick}
                  ref={clearButtonRef}
                >
                  Clear external ID
                </Button>
              ) : (
                <Button
                  type="button"
                  form="edit-external-id-form"
                  variant="primary"
                  size="xlg"
                  className="rounded-lg flex-1"
                  loading={updateKeyOwner.isLoading}
                  onClick={handleSubmit}
                  disabled={!originalIdentityId && !selectedIdentityId}
                >
                  Update external ID
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
        <FormCombobox
          optional
          label="External ID"
          description="ID of the user/workspace in your system for key attribution."
          options={identityOptions}
          value={originalIdentityId || selectedIdentityId || ""}
          onChange={(e) => setSearchValue(e.currentTarget.value)}
          onSelect={(val) => {
            const identity = identities.find((id) => id.id === val);
            setSelectedIdentityId(identity?.id || null);
          }}
          placeholder={
            <div className="flex w-full text-grayA-8 text-xs gap-1.5 items-center py-2">
              Select external ID
            </div>
          }
          searchPlaceholder="Search external ID..."
          emptyMessage={
            <div className="flex flex-col gap-4 items-center justify-center py-2">
              <span className="text-gray-9 text-sm">No external ID found.</span>
              <Button
                variant="outline"
                size="md"
                className="w-fit rounded-lg"
                onClick={handleCreateIdentity}
                loading={createIdentity.isLoading}
                disabled={!searchValue.trim() || createIdentity.isLoading}
              >
                <Plus size="sm-regular" />
                Create new identity
              </Button>
            </div>
          }
          variant="default"
        />
      </DialogContainer>
      <ConfirmPopover
        isOpen={isConfirmPopoverOpen}
        onOpenChange={setIsConfirmPopoverOpen}
        onConfirm={clearSelection}
        triggerRef={clearButtonRef}
        title="Confirm removing external ID"
        description="This will remove the external ID association from this key. Any tracking or analytics related to this ID will no longer be associated with this key."
        confirmButtonText="Remove external ID"
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
};
