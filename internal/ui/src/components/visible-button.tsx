"use client";
import { Eye, EyeSlash } from "@unkey/icons";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import { Button, type ButtonProps } from "./button";

type VisibleButtonProps = ButtonProps & {
  isVisible: boolean;
  setIsVisible: (visible: boolean) => void;
  variant?: ButtonProps["variant"];
};

export function VisibleButton({ isVisible, setIsVisible, variant, ...props }: VisibleButtonProps) {
  const { title, className, ...rest } = props;
  return (
    <Button
      {...rest}
      variant={variant ?? "outline"}
      size="icon"
      className={className}
      onClick={() => setIsVisible(!isVisible)}
      aria-label={isVisible ? `Hide ${title}` : `Show ${title}`}
      title={isVisible ? `Hide ${title}` : `Show ${title}`}
    >
      {isVisible ? <EyeSlash /> : <Eye />}
    </Button>
  );
}
