import { cn } from "@/lib/utils";
import { Badge } from "@unkey/ui";

type SelectedConfigProps = {
  label: string;
  className?: string;
};

export const SelectedConfig = ({ label, className = "" }: SelectedConfigProps) => {
  return (
    <Badge
      variant="secondary"
      className={cn(
        "px-3 py-2 text-gray-11 text-[13px] bg-transparent hover:bg-grayA-3 cursor-default hover:text-gray-12 rounded-md focus:hover:bg-transparent h-7",
        className,
      )}
    >
      {label}
    </Badge>
  );
};
