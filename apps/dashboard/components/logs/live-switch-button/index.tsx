import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { CircleCarretRight } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";

type LiveSwitchProps = {
  isLive: boolean;
  onToggle: () => void;
};

export const LiveSwitchButton = ({ isLive, onToggle }: LiveSwitchProps) => {
  useKeyboardShortcut({ meta: true, key: "l" }, onToggle);

  return (
    <Button
      onClick={onToggle}
      variant="ghost"
      title="Toggle live updates (Shortcut: ⌘ + L)"
      className={cn(
        "px-2 relative",
        isLive
          ? "bg-info-3 text-info-11 hover:bg-info-3 hover:text-info-11 border border-solid border-info-7"
          : "text-accent-12 [&_svg]:text-accent-9",
      )}
    >
      {isLive && (
        <div className="absolute left-0 right-0 top-0 bottom-0 rounded">
          <div className="absolute inset-0 bg-info-6 rounded opacity-15 animate-[ping_3s_cubic-bezier(0,0,0.2,1)_infinite]" />
        </div>
      )}
      <CircleCarretRight className="size-4 relative z-10" />
      <span className="font-medium text-[13px]">Live</span>
    </Button>
  );
};
