import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { ArrowOppositeDirectionY, Ban, CircleCheck, Trash, XMark } from "@unkey/icons";
import { Button, ConfirmPopover } from "@unkey/ui";
import { AnimatePresence, motion } from "framer-motion";
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

  return (
    <>
      <AnimatePresence>
        {selectedKeys.size > 0 && (
          <motion.div
            key="selection-controls"
            className="border-b border-grayA-3 w-full overflow-hidden"
            initial={{ opacity: 0, height: 0 }}
            animate={{
              opacity: 1,
              height: "auto",
              transition: {
                height: { duration: 0.3, ease: "easeOut" },
                opacity: { duration: 0.3, ease: "easeOut" },
              },
            }}
            exit={{
              opacity: 0,
              height: 0,
              transition: {
                opacity: { duration: 0.2, ease: "easeIn" },
                height: { duration: 0.3, ease: "easeIn" },
              },
            }}
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
                  disabled={getSelectedKeysState() !== "all-disabled" || updateKeyStatus.isLoading}
                  loading={updateKeyStatus.isLoading}
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
                  disabled={getSelectedKeysState() !== "all-enabled" || updateKeyStatus.isLoading}
                  loading={updateKeyStatus.isLoading}
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
                  disabled={deleteKey.isLoading}
                  loading={deleteKey.isLoading}
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
          </motion.div>
        )}
      </AnimatePresence>

      <ConfirmPopover
        isOpen={isDisableConfirmOpen}
        onOpenChange={setIsDisableConfirmOpen}
        onConfirm={performDisableKeys}
        triggerRef={disableButtonRef}
        title="Confirm disabling keys"
        description={`This will disable ${selectedKeys.size} key${
          selectedKeys.size > 1 ? "s" : ""
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
        description={`This action is irreversible. All data associated with ${
          selectedKeys.size > 1 ? "these keys" : "this key"
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
    </>
  );
};

export const AnimatedCounter = ({ value }: { value: number }) => {
  return (
    <motion.div
      key={`counter-${value}`}
      initial={{ opacity: 0, scale: 0.5 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.4, ease: "easeOut" }}
      className="size-[18px] text-[11px] leading-6 ring-2 ring-gray-6 flex items-center justify-center font-medium overflow-hidden p-2 text-white dark:text-black bg-accent-12 hover:bg-accent-12/90 focus:hover:bg-accent-12 rounded-md border border-grayA-4"
    >
      <span className="flex items-center justify-center">{value}</span>
    </motion.div>
  );
};
