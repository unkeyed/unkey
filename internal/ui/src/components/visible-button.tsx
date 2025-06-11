"use client";

// TODO: Convert to Nucleo Icons, Add them to unkey/icons
import { Hide, View } from "@unkey/icons";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React, { useEffect } from "react";
import { Button, type ButtonProps } from "./button";
type VisibleButtonProps = ButtonProps & {
  isVisible: boolean;
  setIsVisible: (visible: boolean) => void;
};

export function VisibleButton({
  className,
  isVisible,
  setIsVisible,
  ...props
}: VisibleButtonProps) {
  useEffect(() => {
    if (!isVisible) {
      return;
    }
    const timer = setTimeout(() => {
      setIsVisible(false);
    }, 10000);
    return () => clearTimeout(timer);
  }, [setIsVisible, isVisible]);

  return (
    <Button
      {...props}
      type="button"
      shape="square"
      variant="ghost"
      size="icon"
      className={className}
      onClick={() => {
        setIsVisible(!isVisible);
      }}
    >
      <span className="sr-only">Show</span>
      {isVisible ? <View size="lg-thin" /> : <Hide size="lg-thin" />}
    </Button>
  );
}
