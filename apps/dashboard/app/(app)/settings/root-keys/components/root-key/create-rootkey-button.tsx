"use client";
import { cn } from "@/lib/utils";
import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { ROOT_KEY_MESSAGES } from "./constants";
import { RootKeyDialog } from "./root-key-dialog";

type Props = {
  className?: string;
  triggerRef?: React.RefObject<HTMLButtonElement>;
} & React.ComponentProps<typeof Button>;

export const CreateRootKeyButton = ({ className, ...props }: Props) => {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      <Button
        {...props}
        title={ROOT_KEY_MESSAGES.UI.NEW_ROOT_KEY}
        onClick={() => setIsOpen(true)}
        variant="primary"
        size="md"
        className={cn("rounded-lg", className)}
      >
        <Plus />
        {ROOT_KEY_MESSAGES.UI.NEW_ROOT_KEY}
      </Button>
      <RootKeyDialog
        title={ROOT_KEY_MESSAGES.UI.NEW_ROOT_KEY}
        subTitle="Define a new root key and assign permissions"
        isOpen={isOpen}
        onOpenChange={setIsOpen}
      />
    </>
  );
};
