import { Button } from "@unkey/ui";
import type { PropsWithChildren } from "react";

type QueriesToastProps = PropsWithChildren<{
  message: string;
  undoBookmarked: () => void;
}>;
export const QueriesToast = ({ children, message, undoBookmarked }: QueriesToastProps) => {
  return (
    <div className="flex items-center w-full gap-4 px-1">
      <div className="flex">{children}</div>
      <span className="flex-1 text-sm font-medium leading-6 text-left bg-base-12 w-full">
        {message}
      </span>
      <Button
        variant="ghost"
        className="shrink-0 px-[10px] py-[2px] m-0 w-[54px] h-[28px] rounded-[8px] border border-gray-a6 bg-base-12 text-accent-12"
        onClick={undoBookmarked}
      >
        Undo
      </Button>
    </div>
  );
};
