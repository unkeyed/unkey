"use client";
import { Eye, EyeSlash } from "@unkey/icons";
import { cn } from "../lib/utils";
import { Button, type ButtonProps } from "./button";

type VisibleButtonProps = ButtonProps & {
  isVisible: boolean;
  setIsVisible: (visible: boolean) => void;
};

export function VisibleButton({ isVisible, setIsVisible, ...props }: VisibleButtonProps) {
  const { title, className, ...rest } = props;
  return (
    <Button
      {...rest}
      variant="outline"
      size="icon"
      className={cn("bg-grayA-3 transition-all", className)}
      onClick={() => setIsVisible(!isVisible)}
      aria-label={isVisible ? `Hide ${title}` : `Show ${title}`}
      title={isVisible ? `Hide ${title}` : `Show ${title}`}
    >
      {isVisible ? <EyeSlash /> : <Eye />}
    </Button>
  );
}
