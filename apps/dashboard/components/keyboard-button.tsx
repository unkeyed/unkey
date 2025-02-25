import { Button } from "@unkey/ui";
import type { ComponentProps } from "react";

type ModifierKey = "⌘" | "⇧" | "CTRL" | "⌥";

interface KeyboardButtonProps extends ComponentProps<typeof Button> {
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
    <Button
      variant="ghost"
      tabIndex={-1}
      className={`h-5 px-1.5 min-w-[24px] rounded bg-gray-3 text-gray-9 dark:text-gray-10 border-gray-8 dark:border-gray-9 border text-xs ${className}`}
      aria-label={`Keyboard shortcut ${modifierKey || ""} ${shortcut}`}
      role="presentation"
      aria-haspopup="true"
      title={`Press '${modifierKey ?? ""}${shortcut?.toUpperCase()}' to toggle`}
      {...props}
    >
      <div>
        {modifierKey && <>{modifierKey}+</>}
        {<span className="font-mono">{shortcut.toUpperCase()}</span>}
      </div>
    </Button>
  );
};
