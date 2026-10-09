import { IconTrashOutline18 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "cn";
import type { RefObject } from "react";

type RemoveButtonProps = {
  onClick: () => void;
  className?: string;
  ref?: RefObject<HTMLButtonElement | null>;
  label: string;
  disabled?: boolean;
};

export function RemoveButton({ onClick, className, ref, label, disabled }: RemoveButtonProps) {
  return (
    <Button
      type="button"
      variant="ghost"
      ref={ref}
      aria-label={label}
      disabled={disabled}
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
}
