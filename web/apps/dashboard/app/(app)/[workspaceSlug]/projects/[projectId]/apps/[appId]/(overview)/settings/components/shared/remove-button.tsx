import { Button } from "@unkey/ui";
import { IconTrashOutline18 } from "nucleo-ui-outline-18";
import type { RefObject } from "react";
import { cn } from "@/lib/utils";

type RemoveButtonProps = {
  onClick: () => void;
  className?: string;
  ref?: RefObject<HTMLButtonElement | null>;
};

export const RemoveButton = ({ onClick, className, ref }: RemoveButtonProps) => (
  <Button
    type="button"
    variant="ghost"
    ref={ref}
    size="sm"
    className={cn(
      "size-9 px-0 justify-center text-error-11 hover:text-error-11 shrink-0 hover:bg-grayA-3 rounded-lg",
      className,
    )}
    onClick={onClick}
  >
    <IconTrashOutline18 />
  </Button>
);
