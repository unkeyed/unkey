"use client";

import { IconCircleLockOutline18 } from "@unkey/icons";
import { CopyButton, InputGroup, InputGroupAddon, InputGroupInput, VisibleButton } from "@unkey/ui";
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
    <InputGroup className={cn("min-w-0 unkey-root-key", className)}>
      <InputGroupAddon>
        <IconCircleLockOutline18 className="size-3 text-gray-12" />
      </InputGroupAddon>
      <InputGroupInput
        readOnly
        value={displayValue}
        aria-label={title}
        className="truncate font-mono"
      />
      <InputGroupAddon align="inline-end">
        <VisibleButton
          isVisible={isVisible}
          setIsVisible={(visible) => setIsVisible(visible)}
          title={title}
        />
        <CopyButton value={value} title={title} />
      </InputGroupAddon>
    </InputGroup>
  );
};
