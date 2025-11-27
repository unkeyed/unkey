import { ConfirmPopover } from "@/components/confirmation-popover";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { ArrowOppositeDirectionY, Ban, CircleCheck, Trash, XMark } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useRef, useState } from "react";
import { useDeleteKey } from "../actions/components/hooks/use-delete-key";
import { useBatchUpdateKeyStatus } from "../actions/components/hooks/use-update-key-status";
import { BatchEditExternalId } from "./components/batch-edit-external-id";

type SelectionControlsProps = {
  selectedKeys: Set<string>;
  setSelectedKeys: (keys: Set<string>) => void;
  keys: KeyDetails[];
  getSelectedKeysState: () => "all-enabled" | "all-disabled" | "mixed" | null;
};

export const SelectionControls = ({
  selectedKeys,
  keys,
  setSelectedKeys,
  getSelectedKeysState,
}: SelectionControlsProps) => {
  const [isBatchEditExternalIdOpen, setIsBatchEditExternalIdOpen] = useState(false);
  const [isDisableConfirmOpen, setIsDisableConfirmOpen] = useState(false);
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);

  const disableButtonRef = useRef<HTMLButtonElement>(null);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  const updateKeyStatus = useBatchUpdateKeyStatus();
  const deleteKey = useDeleteKey(() => {
    setSelectedKeys(new Set());
  });

  const handleDisableButtonClick = () => {
    setIsDisableConfirmOpen(true);
  };

  const performDisableKeys = () => {
    updateKeyStatus.mutate({
      enabled: false,
      keyIds: Array.from(selectedKeys),
    });
  };

  const handleDeleteButtonClick = () => {
    setIsDeleteConfirmOpen(true);
  };

  const performKeyDeletion = () => {
    deleteKey.mutate({
      keyIds: Array.from(selectedKeys),
    });
  };

  const keysWithExternalIds = keys.filter(
    (key) => selectedKeys.has(key.id) && key.identity_id,
  ).length;

  if (selectedKeys.size === 0) {
    return null;
  }

  return (
    <>
      <div
        className={cn(
          "border-b border-grayA-3",
          "animate-slideInFromTop opacity-0 translate-y-2",
          "animation-fill-mode-forwards",
        )}
      >
        <div className="flex justify-between items-center w-full p-[18px]">
          <div className="items-center flex gap-2">
            <AnimatedCounter value={selectedKeys.size} />
            <div className="text-accent-9 text-[13px] leading-6">selected</div>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              className="text-gray-12 font-medium text-[13px]"
              onClick={() => setIsBatchEditExternalIdOpen(true)}
            >
              <ArrowOppositeDirectionY iconSize="sm-regular" /> Change External ID
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="text-gray-12 font-medium text-[13px]"
              disabled={getSelectedKeysState() !== "all-disabled" || updateKeyStatus.isPending}
              loading={updateKeyStatus.isPending}
              onClick={() =>
                updateKeyStatus.mutate({
                  enabled: true,
                  keyIds: Array.from(selectedKeys),
                })
              }
            >
              <CircleCheck iconSize="sm-regular" />
              Enable key
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="text-gray-12 font-medium text-[13px]"
              disabled={getSelectedKeysState() !== "all-enabled" || updateKeyStatus.isPending}
              loading={updateKeyStatus.isPending}
              onClick={handleDisableButtonClick}
              ref={disableButtonRef}
            >
              <Ban iconSize="sm-regular" />
              Disable key
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="text-gray-12 font-medium text-[13px]"
              disabled={deleteKey.isPending}
              loading={deleteKey.isPending}
              onClick={handleDeleteButtonClick}
              ref={deleteButtonRef}
            >
              <Trash iconSize="sm-regular" />
              Delete key
            </Button>
            <Button
              size="icon"
              variant="ghost"
              className="[&_svg]:size-[14px] ml-3"
              onClick={() => setSelectedKeys(new Set())}
            >
              <XMark />
            </Button>
          </div>
        </div>
      </div>

      <ConfirmPopover
        isOpen={isDisableConfirmOpen}
        onOpenChange={setIsDisableConfirmOpen}
        onConfirm={performDisableKeys}
        triggerRef={disableButtonRef}
        title="Confirm disabling keys"
        description={`This will disable ${selectedKeys.size} key${selectedKeys.size > 1 ? "s" : ""
          } and prevent any verification requests from being processed.`}
        confirmButtonText="Disable keys"
        cancelButtonText="Cancel"
        variant="danger"
      />

      <ConfirmPopover
        isOpen={isDeleteConfirmOpen}
        onOpenChange={setIsDeleteConfirmOpen}
        onConfirm={performKeyDeletion}
        triggerRef={deleteButtonRef}
        title="Confirm key deletion"
        description={`This action is irreversible. All data associated with ${selectedKeys.size > 1 ? "these keys" : "this key"
          } will be permanently deleted.`}
        confirmButtonText={`Delete key${selectedKeys.size > 1 ? "s" : ""}`}
        cancelButtonText="Cancel"
        variant="danger"
      />

      {isBatchEditExternalIdOpen && (
        <BatchEditExternalId
          selectedKeyIds={Array.from(selectedKeys)}
          keysWithExternalIds={keysWithExternalIds}
          isOpen={isBatchEditExternalIdOpen}
          onClose={() => setIsBatchEditExternalIdOpen(false)}
        />
      )}

      <style jsx>{`
        @keyframes slideInFromTop {
          from {
            opacity: 0;
            transform: translateY(-10px);
          }
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }

        @keyframes bounceIn {
          0% {
            opacity: 0;
            transform: scale(0.5);
          }
          50% {
            transform: scale(1.1);
          }
          100% {
            opacity: 1;
            transform: scale(1);
          }
        }

        .animate-slideInFromTop {
          animation: slideInFromTop 0.3s ease-out;
        }

        .animate-bounceIn {
          animation: bounceIn 0.4s ease-out;
        }

        .animation-fill-mode-forwards {
          animation-fill-mode: forwards;
        }
      `}</style>
    </>
  );
};

export const AnimatedCounter = ({ value }: { value: number }) => {
  return (
    <div
      key={`counter-${value}`}
      className={cn(
        "size-[18px] text-[11px] leading-6 ring-2 ring-gray-6 flex items-center justify-center font-medium overflow-hidden p-2 text-white dark:text-black bg-accent-12 hover:bg-accent-12/90 focus:hover:bg-accent-12 rounded-md border border-grayA-4",
        "animate-bounceIn",
      )}
    >
      <span className="flex items-center justify-center">{value}</span>
    </div>
  );
};
