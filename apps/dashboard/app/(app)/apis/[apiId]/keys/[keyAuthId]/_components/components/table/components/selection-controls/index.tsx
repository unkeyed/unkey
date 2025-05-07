import { ConfirmPopover } from "@/components/confirmation-popover";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { ArrowOppositeDirectionY, Ban, CircleCheck, Trash, XMark } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { AnimatePresence, motion } from "framer-motion";
import { useRef, useState } from "react";
import { useBatchUpdateKeyStatus } from "../actions/components/hooks/use-update-key-status";

type SelectionControlsProps = {
  selectedKeys: Set<string>;
  setSelectedKeys: (keys: Set<string>) => void;
  keys: KeyDetails[];
  getSelectedKeysState: () => "all-enabled" | "all-disabled" | "mixed" | null;
};

export const SelectionControls: React.FC<SelectionControlsProps> = ({
  selectedKeys,
  setSelectedKeys,
  getSelectedKeysState,
}) => {
  const updateKeyStatus = useBatchUpdateKeyStatus();
  const [isConfirmPopoverOpen, setIsConfirmPopoverOpen] = useState(false);
  const disableButtonRef = useRef<HTMLButtonElement>(null);

  const handleDisableKeys = () => {
    setIsConfirmPopoverOpen(true);
  };

  const performDisableKeys = () => {
    updateKeyStatus.mutate({
      enabled: false,
      keyIds: Array.from(selectedKeys),
    });
  };

  return (
    <AnimatePresence>
      {selectedKeys.size > 0 ? (
        <div className="border-b border-grayA-3">
          <motion.div
            key="selection-controls"
            className="flex justify-between items-center w-full p-[18px]"
            initial={{ opacity: 0, y: 10 }}
            animate={{
              opacity: 1,
              y: 0,
              transition: {
                opacity: { duration: 0.3, ease: "easeOut" },
                y: { duration: 0.3, ease: "easeOut" },
              },
            }}
            exit={{
              opacity: 0,
              y: 10,
              transition: {
                opacity: { duration: 0.3, ease: "easeIn" },
                y: { duration: 0.3, ease: "easeIn" },
              },
            }}
          >
            <div className="items-center flex gap-2">
              <Button
                variant="primary"
                className="size-[18px] text-[11px] leading-6 ring-2 ring-gray-6 rounded-md"
              >
                {selectedKeys.size}
              </Button>
              <div className="text-accent-9 text-[13px] leading-6">selected</div>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" className="text-gray-12 font-medium text-[13px]">
                <ArrowOppositeDirectionY size="sm-regular" /> Change External ID
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
                <CircleCheck size="sm-regular" />
                Enable key
              </Button>
              <Button
                variant="outline"
                size="sm"
                className="text-gray-12 font-medium text-[13px]"
                disabled={getSelectedKeysState() !== "all-enabled" || updateKeyStatus.isLoading}
                loading={updateKeyStatus.isLoading}
                onClick={handleDisableKeys}
                ref={disableButtonRef}
              >
                <Ban size="sm-regular" />
                Disable key
              </Button>
              <Button variant="outline" size="sm" className="text-gray-12 font-medium text-[13px]">
                <Trash size="sm-regular" />
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
          </motion.div>
        </div>
      ) : null}

      <ConfirmPopover
        isOpen={isConfirmPopoverOpen}
        onOpenChange={setIsConfirmPopoverOpen}
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
    </AnimatePresence>
  );
};
