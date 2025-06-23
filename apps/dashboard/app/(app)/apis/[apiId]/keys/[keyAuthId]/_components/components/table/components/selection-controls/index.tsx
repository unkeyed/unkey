import { ConfirmPopover } from "@/components/confirmation-popover";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { ArrowOppositeDirectionY, Ban, CircleCheck, Trash, XMark } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useEffect, useRef, useState } from "react";
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
  const [isVisible, setIsVisible] = useState(false);
  const [shouldRender, setShouldRender] = useState(false);

  const disableButtonRef = useRef<HTMLButtonElement>(null);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  const updateKeyStatus = useBatchUpdateKeyStatus();
  const deleteKey = useDeleteKey(() => {
    setSelectedKeys(new Set());
  });

  // Handle show/hide animations
  useEffect(() => {
    if (selectedKeys.size > 0) {
      setShouldRender(true);
      // Trigger animation after render
      requestAnimationFrame(() => {
        setIsVisible(true);
      });
    } else {
      setIsVisible(false);
      // Clean up after exit animation
      const timer = setTimeout(() => {
        setShouldRender(false);
      }, 300);
      return () => clearTimeout(timer);
    }
  }, [selectedKeys.size]);

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

  if (!shouldRender) {
    return null;
  }

  return (
    <>
      <div
        className={`border-b border-grayA-3 transition-all duration-300 ease-out ${
          isVisible ? "opacity-100 translate-y-0" : "opacity-0 translate-y-2"
        }`}
        style={{
          animation: isVisible ? "slideInFromTop 0.3s ease-out" : undefined,
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
              onClick={handleDisableButtonClick}
              ref={disableButtonRef}
            >
              <Ban size="sm-regular" />
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
        </div>
      </div>

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
            transform: scale(0.3) translateY(-20px);
          }
          50% {
            opacity: 1;
            transform: scale(1.05) translateY(0);
          }
          70% {
            transform: scale(0.95);
          }
          100% {
            opacity: 1;
            transform: scale(1);
          }
        }

        @keyframes digitSlideIn {
          from {
            opacity: 0;
            transform: translateY(-10px);
          }
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }
      `}</style>
    </>
  );
};

export const AnimatedCounter = ({ value }: { value: number }) => {
  const [prevValue, setPrevValue] = useState(value);
  const [shouldAnimate, setShouldAnimate] = useState(false);

  useEffect(() => {
    if (value !== prevValue) {
      setShouldAnimate(true);
      setPrevValue(value);

      const timer = setTimeout(() => {
        setShouldAnimate(false);
      }, 600);

      return () => clearTimeout(timer);
    }
  }, [value, prevValue]);

  return (
    <div
      className={`size-[18px] text-[11px] leading-6 ring-2 ring-gray-6 flex items-center justify-center font-medium overflow-hidden
p-2 text-white dark:text-black bg-accent-12 hover:bg-accent-12/90 focus:hover:bg-accent-12 rounded-md border border-grayA-4 
transition-all duration-200 ${shouldAnimate ? "scale-110" : "scale-100"}`}
      style={{
        animation: shouldAnimate ? "bounceIn 0.6s ease-out" : undefined,
      }}
    >
      <span
        key={value}
        className="flex items-center justify-center"
        style={{
          animation: shouldAnimate ? "digitSlideIn 0.3s ease-out" : undefined,
        }}
      >
        {value}
      </span>
    </div>
  );
};
