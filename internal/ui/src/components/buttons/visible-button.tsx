"use client";
import { Eye, EyeSlash } from "@unkey/icons";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import { cn } from "../../lib/utils";
import { Button, type ButtonProps } from "./button";

type VisibleButtonProps = ButtonProps & {
  /**
   * Current visibility state
   */
  isVisible: boolean;
  /**
   * Function to set the visibility state
   */
  setIsVisible: (visible: boolean) => void;
  /**
   * Variant of the button
   */
  variant?: ButtonProps["variant"];
};

function VisibleButton({
  isVisible,
  setIsVisible,
  variant = "outline",
  title,
  onClick,
  className,
  ...rest
}: VisibleButtonProps) {
  return (
    <Button
      {...rest}
      type="button"
      title={isVisible ? `Hide ${title}` : `Show ${title}`}
      variant={variant}
      className={cn("focus:ring-0 focus:border-grayA-6 w-6 h-6", className)}
      onClick={(e) => {
        if (!e.defaultPrevented) {
          setIsVisible(!isVisible);
        }
      }}
      aria-label={isVisible ? `Hide ${title}` : `Show ${title}`}
    >
      {isVisible ? <EyeSlash /> : <Eye />}
    </Button>
  );
}

VisibleButton.displayName = "VisibleButton";

export { VisibleButton, type VisibleButtonProps };
