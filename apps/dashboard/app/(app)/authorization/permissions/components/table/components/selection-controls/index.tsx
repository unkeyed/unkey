import { AnimatedCounter } from "@/app/(app)/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/selection-controls";
import { ConfirmPopover } from "@/components/confirmation-popover";
import { Trash, XMark } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { AnimatePresence, motion } from "framer-motion";
import { useRef, useState } from "react";
import { useDeletePermission } from "../actions/components/hooks/use-delete-permission";

type SelectionControlsProps = {
  selectedPermissions: Set<string>;
  setSelectedPermissions: (keys: Set<string>) => void;
};

export const SelectionControls = ({
  selectedPermissions,
  setSelectedPermissions,
}: SelectionControlsProps) => {
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  const deletePermission = useDeletePermission(() => {
    setSelectedPermissions(new Set());
  });

  const handleDeleteButtonClick = () => {
    setIsDeleteConfirmOpen(true);
  };

  const performPermissionDelete = () => {
    deletePermission.mutate({
      permissionIds: Array.from(selectedPermissions),
    });
  };

  return (
    <>
      <AnimatePresence>
        {selectedPermissions.size > 0 && (
          <motion.div
            key="selection-controls"
            className="border-b border-grayA-3"
            initial={{ opacity: 0, y: 10 }}
            animate={{
              opacity: 1,
              y: 5,
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
            <div className="flex justify-between items-center w-full p-[18px]">
              <div className="items-center flex gap-2">
                <AnimatedCounter value={selectedPermissions.size} />
                <div className="text-accent-9 text-[13px] leading-6">selected</div>
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  className="text-gray-12 font-medium text-[13px]"
                  disabled={deletePermission.isLoading}
                  loading={deletePermission.isLoading}
                  onClick={handleDeleteButtonClick}
                  ref={deleteButtonRef}
                >
                  <Trash size="sm-regular" />
                  Delete permissions
                </Button>
                <Button
                  size="icon"
                  variant="ghost"
                  className="[&_svg]:size-[14px] ml-3"
                  onClick={() => setSelectedPermissions(new Set())}
                >
                  <XMark />
                </Button>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      <ConfirmPopover
        isOpen={isDeleteConfirmOpen}
        onOpenChange={setIsDeleteConfirmOpen}
        onConfirm={performPermissionDelete}
        triggerRef={deleteButtonRef}
        title="Confirm permission deletion"
        description={`This action is irreversible. All data associated with ${
          selectedPermissions.size > 1 ? "these permissions" : "this permission"
        } will be permanently deleted.`}
        confirmButtonText={`Delete permission${selectedPermissions.size > 1 ? "s" : ""}`}
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
};
