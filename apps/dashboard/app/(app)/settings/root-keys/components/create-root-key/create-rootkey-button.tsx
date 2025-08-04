"use client";
import { cn } from "@/lib/utils";
import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
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
        title="New root key"
        onClick={() => setIsOpen(true)}
        variant="primary"
        size="md"
        className={cn("rounded-lg", className)}
      >
        <Plus />
        New root key
      </Button>
      <RootKeyDialog
        title="New root key"
        subTitle="Define a new root key and assign permissions"
        isOpen={isOpen}
        onOpenChange={setIsOpen}
      />
    </>
  );
};
