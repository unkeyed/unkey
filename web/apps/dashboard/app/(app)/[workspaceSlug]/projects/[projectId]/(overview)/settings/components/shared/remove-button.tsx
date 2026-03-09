import { cn } from "@/lib/utils";
import { Trash } from "@unkey/icons";
import { Button } from "@unkey/ui";

type RemoveButtonProps = {
  onClick: () => void;
  className?: string;
};

export const RemoveButton = ({ onClick, className }: RemoveButtonProps) => (
  <Button
    type="button"
    variant="ghost"
    size="sm"
    className={cn(
      "size-9 px-0 justify-center text-error-11 hover:text-error-11 shrink-0 hover:bg-grayA-3 rounded-lg",
      className,
    )}
    onClick={onClick}
  >
    <Trash iconSize="sm-regular" />
  </Button>
);
