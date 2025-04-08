import { cn } from "@/lib/utils";
import type { ComponentProps } from "react";

type ModifierKey = "⌘" | "⇧" | "CTRL" | "⌥";

interface KeyboardButtonProps extends ComponentProps<"div"> {
  shortcut: string;
  modifierKey?: ModifierKey | null;
}

export const KeyboardButton = ({
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
        { className },
      )}
      aria-label={`Keyboard shortcut ${modifierKey || ""} ${shortcut}`}
      role="presentation"
      aria-haspopup="true"
      title={`Press '${modifierKey ?? ""}${shortcut?.toUpperCase()}' to toggle`}
      {...props}
    >
      {modifierKey && <kbd>{modifierKey}+</kbd>}
      <kbd>{shortcut.toUpperCase()}</kbd>
    </span>
  );
};
