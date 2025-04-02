import { Button } from "@unkey/ui";
import type { PropsWithChildren } from "react";

type QueriesToastProps = PropsWithChildren<{
  message: string;
  undoBookmarked: () => void;
}>;
export const QueriesToast = ({ children, message, undoBookmarked }: QueriesToastProps) => {
  return (
    <div className="flex flex-row items-center justify-center w-full gap-4 px-1 shrink">
      <div>{children}</div>
      <span className="flex justify-start w-full text-sm font-medium leading-6 text-left bg-base-12">
        {message}
      </span>
      <Button
        variant="ghost"
        className="flex end-0 px-[10px] py-[2px] m-0 w-[54px] h-[28px] rounded-[8px] border-[1px] border-gray-a6 bg-base-12 text-accent-12"
        onClick={undoBookmarked}
      >
        Undo
      </Button>
    </div>
  );
};
