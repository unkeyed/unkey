"use client";

import { IconCircleLockOutline18 } from "@unkey/icons";
import { CopyButton, VisibleButton } from "@unkey/ui";
import { cn } from "cn";
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
        "min-w-0 w-full px-4 py-2 bg-raised border rounded-xl unkey-root-key",
        className,
      )}
    >
      <div className="flex items-center justify-between w-full gap-3 pointer-events-auto">
        <div className="shrink-0">
          <IconCircleLockOutline18 className="size-3 text-gray-12" />
        </div>
        <div className="flex-1 min-w-0 overflow-hidden">
          <p className="truncate font-mono text-sm text-grayA-12 pr-2">{displayValue}</p>
        </div>
        <div className="flex items-center justify-between gap-2 shrink-0 pointer-events-auto">
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
