"use client";
import { cn } from "@/lib/utils";
import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import type { RefObject } from "react";
import { useState } from "react";
import { ROOT_KEY_MESSAGES } from "./constants";
import { RootKeyDialog } from "./root-key-dialog";

type Props = {
  className?: string;
  triggerRef?: RefObject<HTMLButtonElement | null>;
} & React.ComponentProps<typeof Button>;

const CreateRootKeyButton = ({ className, triggerRef, onClick, ...props }: Props) => {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="relative">
      <Button
        {...props}
        title={ROOT_KEY_MESSAGES.UI.NEW_ROOT_KEY}
        onClick={(e) => {
          onClick?.(e);
          setIsOpen(true);
        }}
        ref={triggerRef}
        variant="primary"
        type="button"
        size="sm"
        className={cn("px-3 rounded-md", className)}
      >
        <Plus />
        {ROOT_KEY_MESSAGES.UI.NEW_ROOT_KEY}
      </Button>
      <RootKeyDialog
        title={ROOT_KEY_MESSAGES.UI.NEW_ROOT_KEY}
        subTitle={ROOT_KEY_MESSAGES.UI.NEW_ROOT_KEY_SUBTITLE}
        isOpen={isOpen}
        onOpenChange={setIsOpen}
      />
    </div>
  );
};

CreateRootKeyButton.displayName = "CreateRootKeyButton";

export { CreateRootKeyButton };
