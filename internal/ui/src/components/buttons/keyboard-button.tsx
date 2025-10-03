"use client";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import type { ComponentProps } from "react";
import { cn } from "../../lib/utils";

type ModifierKey = "⌘" | "⇧" | "CTRL" | "⌥";

interface KeyboardButtonProps extends ComponentProps<"span"> {
  shortcut: string;
  modifierKey?: ModifierKey | null;
}

const KeyboardButton = ({
  shortcut,
  modifierKey,
  className = "",
  ...props
}: KeyboardButtonProps) => {
  return (
    <span
      tabIndex={-1}
      className={cn(
        "flex items-center justify-center h-5 px-1 min-w-[24px] bg-secondary rounded bg-gray-3 text-gray-9 dark:text-gray-10 border-gray-8 dark:border-gray-9 border text-xs",
        "p-2 text-gray-12 hover:bg-grayA-4 rounded-md focus:hover:bg-transparent",
        "focus:border-grayA-12 focus:ring-4 focus:ring-gray-6 focus-visible:outline-none focus:ring-offset-0 drop-shadow-button",
        "disabled:border disabled:border-grayA-4 disabled:text-grayA-7",
        "active:bg-grayA-5 max-md:hidden",
        className,
      )}
      aria-label={`Keyboard shortcut ${modifierKey || ""} ${shortcut}`}
      role="presentation"
      aria-haspopup="true"
      title={`Press '${modifierKey ?? ""}${shortcut?.toUpperCase()}' to toggle`}
      {...props}
    >
      {/* className="not-prose" added to prevent markdown rendering issues */}
      {modifierKey && <kbd className="not-prose">{modifierKey}+</kbd>}
      <kbd className="not-prose">{shortcut?.toUpperCase()}</kbd>
    </span>
  );
};

KeyboardButton.displayName = "KeyboardButton";

export { KeyboardButton };
