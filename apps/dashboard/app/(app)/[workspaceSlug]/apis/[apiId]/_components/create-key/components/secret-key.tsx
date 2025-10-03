"use client"; // Keep if needed

import { cn } from "@/lib/utils";
import { CircleLock } from "@unkey/icons";
import { CopyButton, VisibleButton } from "@unkey/ui";
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

  return (
    <div
      className={cn(
        "w-full px-4 py-2 bg-white dark:bg-black border rounded-xl border-grayA-5",
        className,
      )}
    >
      <div className="flex items-center justify-between w-full gap-3 pointer-events-auto">
        <div className="flex-shrink-0">
          <CircleLock iconsize="sm-regular" className="text-gray-12" />
        </div>
        <div className="flex-1 overflow-x-auto min-w-0">
          {" "}
          <p className="whitespace-pre-wrap break-all font-mono text-[13px] text-grayA-12 pr-2">
            {displayValue}
          </p>
        </div>
        <div className="flex items-center justify-between gap-2 flex-shrink-0 pointer-events-auto">
          <VisibleButton
            isVisible={isVisible}
            setIsVisible={(visible) => setIsVisible(visible)}
            title={title}
          />

          <CopyButton value={value} title={title} />
        </div>
      </div>
    </div>
  );
};
