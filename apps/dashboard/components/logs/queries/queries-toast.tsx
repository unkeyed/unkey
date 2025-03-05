import { Button } from "@unkey/ui";
import type { PropsWithChildren } from "react";

type QueriesToastProps = PropsWithChildren<{
  message: string;
  undoBookmarked: () => void;
}>;
export const QueriesToast = ({ children, message, undoBookmarked }: QueriesToastProps) => {
  return (
    <div className="flex flex-row items-center justify-center w-full gap-4 p-2 font-sans">
      {children}
      <span className="flex items-center justify-center w-56 text-sm font-medium leading-6 text-center text-accent-12">
        {message}
      </span>
      <Button
        variant="ghost"
        className="flex end-0 p-2 m-0  rounded-[10px] border-[1px] border-gray-5"
        onClick={undoBookmarked}
      >
        Undo
      </Button>
    </div>
  );
};
