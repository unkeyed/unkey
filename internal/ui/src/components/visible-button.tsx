"use client";
import { Eye, EyeSlash } from "@unkey/icons";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import { Button, type ButtonProps } from "./button";
import { cn } from "../lib/utils";

type VisibleButtonProps = ButtonProps & {
  isVisible: boolean;
  setIsVisible: (visible: boolean) => void;
  variant?: ButtonProps["variant"];
  size?: ButtonProps["size"];
};

export function VisibleButton({ 
  isVisible,
  setIsVisible, 
  variant = "outline",
  title,
  size = "icon",
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
      size={size}
      className={cn("focus:ring-0 focus:border-grayA-6", className)}
      onClick={(e) => {
        setIsVisible(!isVisible);
        onClick?.(e); 
      }}
      aria-label={isVisible ? `Hide ${title}` : `Show ${title}`}
    >
      {isVisible ? <EyeSlash /> : <Eye />}
    </Button>
  );
}
