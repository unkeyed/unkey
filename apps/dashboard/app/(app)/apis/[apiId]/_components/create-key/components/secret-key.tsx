"use client"; // Keep if needed

import { CopyButton } from "@/components/dashboard/copy-button";
import { cn } from "@/lib/utils";
import { CircleLock, Eye, EyeSlash } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";

const maskKey = (key: string): string => {
  return "•".repeat(key.length);
};

export const SecretKey = ({
  value,
  title = "Value",
  className,
}: {
  value: string;
  title: string;
  className?: string;
}) => {
  const [isVisible, setIsVisible] = useState(false);

  const displayValue = isVisible ? value : maskKey(value);

  const handleToggleVisibility = (e: React.MouseEvent) => {
    e.stopPropagation();
    setIsVisible(!isVisible);
  };

  return (
    <div
      className={cn(
        "w-full px-4 py-2 bg-white dark:bg-black border rounded-xl border-grayA-5",
        className,
      )}
    >
      <div className="flex items-center justify-between w-full gap-3">
        <div className="flex-shrink-0">
          <CircleLock size="sm-regular" className="text-gray-12" />
        </div>
        <div className="flex-1 overflow-x-auto min-w-0">
          {" "}
          <p className="whitespace-pre-wrap break-all font-mono text-[13px] text-grayA-12 pr-2">
            {displayValue}
          </p>
        </div>
        <div className="flex items-center justify-between gap-2 flex-shrink-0">
          <Button
            variant="outline"
            size="icon"
            className="bg-grayA-3 transition-all"
            onClick={handleToggleVisibility}
            aria-label={isVisible ? `Hide ${title}` : `Show ${title}`}
            title={isVisible ? `Hide ${title}` : `Show ${title}`}
          >
            {isVisible ? <EyeSlash /> : <Eye />}
          </Button>

          <Button
            variant="outline"
            size="icon"
            className="bg-grayA-3"
            aria-label={`Copy ${title}`}
            title={`Copy ${title}`}
          >
            <div className="flex items-center justify-center">
              <CopyButton value={value} />
            </div>
          </Button>
        </div>
      </div>
    </div>
  );
};
