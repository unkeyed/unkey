"use client";

import { Button, type ButtonProps } from "@unkey/ui";
import * as React from "react";

// React Loadable props that should be filtered out
interface ReactLoadableProps {
  isLoading?: boolean;
  pastDelay?: boolean;
  timedOut?: boolean;
  retry?: () => void;
  error?: Error | null;
}

export interface NavbarActionButtonProps
  extends Omit<ButtonProps, "title">,
    Partial<ReactLoadableProps> {
  /**
   * The title text to display on the button
   */
  title?: string;
  /**
   * Any content to render on the right side of the button
   */
  rightContent?: React.ReactNode;

  /**
   * Any content to render on the left side of the button
   */
  leftContent?: React.ReactNode;
}

export const NavbarActionButton = React.forwardRef<HTMLButtonElement, NavbarActionButtonProps>(
  (
    {
      title = "Create New Root Key",
      variant = "primary",
      size = "sm",
      className = "px-3",
      disabled,
      loading,
      rightContent,
      leftContent,
      children,
      ...props
    },
    ref,
  ) => {
    // Filter out React Loadable props that shouldn't be passed to DOM elements
    const { isLoading, pastDelay, timedOut, retry, error, ...buttonProps } = props;
    return (
      <div className="flex flex-col gap-1">
        <Button
          ref={ref}
          variant={variant}
          size={size}
          className={className}
          disabled={disabled}
          loading={loading}
          {...buttonProps}
          title={title}
        >
          {leftContent}
          {children}
          {rightContent}
        </Button>
      </div>
    );
  },
);

NavbarActionButton.displayName = "NavbarActionButton";
