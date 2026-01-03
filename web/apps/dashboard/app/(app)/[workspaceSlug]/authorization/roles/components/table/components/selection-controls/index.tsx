import { AnimatedCounter } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/selection-controls";
import { Trash, XMark } from "@unkey/icons";
import { Button, ConfirmPopover } from "@unkey/ui";
import { AnimatePresence, motion } from "framer-motion";
import { useRef, useState } from "react";
import { useDeleteRole } from "../actions/components/hooks/use-delete-role";

type SelectionControlsProps = {
  selectedRoles: Set<string>;
  setSelectedRoles: (keys: Set<string>) => void;
};

export const SelectionControls = ({ selectedRoles, setSelectedRoles }: SelectionControlsProps) => {
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  const deleteRole = useDeleteRole(() => {
    setSelectedRoles(new Set());
  });

  const handleDeleteButtonClick = () => {
    setIsDeleteConfirmOpen(true);
  };

  const performRoleDeletion = () => {
    deleteRole.mutate({
      roleIds: Array.from(selectedRoles),
    });
  };

  return (
    <>
      <AnimatePresence>
        {selectedRoles.size > 0 && (
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
                <AnimatedCounter value={selectedRoles.size} />
                <div className="text-accent-9 text-[13px] leading-6">selected</div>
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  className="text-gray-12 font-medium text-[13px]"
                  disabled={deleteRole.isLoading}
                  loading={deleteRole.isLoading}
                  onClick={handleDeleteButtonClick}
                  ref={deleteButtonRef}
                >
                  <Trash iconSize="sm-regular" />
                  Delete roles
                </Button>
                <Button
                  size="icon"
                  variant="ghost"
                  className="[&_svg]:size-[14px] ml-3"
                  onClick={() => setSelectedRoles(new Set())}
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
        onConfirm={performRoleDeletion}
        triggerRef={deleteButtonRef}
        title="Confirm role deletion"
        description={`This action is irreversible. All data associated with ${
          selectedRoles.size > 1 ? "these roles" : "this role"
        } will be permanently deleted.`}
        confirmButtonText={`Delete role${selectedRoles.size > 1 ? "s" : ""}`}
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
};
