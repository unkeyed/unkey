import * as React from "react";

import { cn } from "@/lib/utils";
import { Eye, EyeOff } from "lucide-react";

type VisibleButtonProps = React.HTMLAttributes<HTMLButtonElement> & {
  isVisible: boolean;
  setIsVisible: (visible: boolean) => void;
};

export function VisibleButton({
  className,
  isVisible,
  setIsVisible,
  ...props
}: VisibleButtonProps) {
  const timeOutRef = React.useRef<NodeJS.Timeout | null>(null);

  React.useEffect(() => {
    const clearVisibilityTimeout = () => {
      if (timeOutRef.current) {
        clearTimeout(timeOutRef.current);
        timeOutRef.current = null;
      }
    };

    clearVisibilityTimeout();

    if (isVisible) {
      const timeOutId = setTimeout(() => {
        setIsVisible(false);
      }, 10000);

      timeOutRef.current = timeOutId;
    }
    return () => {
      clearVisibilityTimeout();
    };
  }, [isVisible]);

  return (
    <button
      type="button"
      className={cn(
        "relative z-20 inline-flex h-8 items-center justify-center rounded-md border-gray-200 p-2 text-sm font-medium text-gray-900 transition-all hover:bg-gray-100 focus:outline-none dark:text-gray-100 dark:hover:bg-gray-800",
        className,
      )}
      onClick={() => {
        setIsVisible(!isVisible);
      }}
      {...props}
    >
      <span className="sr-only">Show</span>
      {isVisible ? <Eye className="w-4 h-4" /> : <EyeOff className="w-4 h-4" />}
    </button>
  );
}
