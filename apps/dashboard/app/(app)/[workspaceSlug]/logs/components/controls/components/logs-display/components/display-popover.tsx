import { isDisplayProperty, useLogsContext } from "@/app/(app)/[workspaceSlug]/logs/context/logs";
import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { KeyboardButton, Popover, PopoverContent, PopoverTrigger } from "@unkey/ui";
import {
  type KeyboardEvent,
  type PropsWithChildren,
  useCallback,
  useEffect,
  useState,
} from "react";

type DisplayPropertyId =
  | "time"
  | "response_status"
  | "method"
  | "path"
  | "response_body"
  | "request_id"
  | "host"
  | "request_headers"
  | "request_body"
  | "response_headers";

type DisplayProperty = {
  id: DisplayPropertyId;
  label: string;
};

const DISPLAY_PROPERTIES: DisplayProperty[] = [
  { id: "time", label: "Time" },
  { id: "response_status", label: "Status" },
  { id: "method", label: "Method" },
  { id: "path", label: "Path" },
  { id: "response_body", label: "Response Body" },
  { id: "request_id", label: "Request ID" },
  { id: "host", label: "Host" },
  { id: "request_headers", label: "Request Headers" },
  { id: "request_body", label: "Request Body" },
  { id: "response_headers", label: "Response Headers" },
];

const DisplayPropertyItem = ({
  label,
  selected,
  onClick,
  isFocused,
  index,
}: {
  label: string;
  selected: boolean;
  onClick: () => void;
  isFocused: boolean;
  index: number;
}) => (
  <div
    data-item-index={index}
    className={`font-medium text-xs p-1.5 rounded-md hover:bg-gray-4 cursor-pointer whitespace-nowrap
      ${selected ? "bg-gray-4 text-gray-12" : "text-gray-9"}
      ${isFocused ? "ring-2 ring-accent-7" : ""}`}
    onClick={onClick}
    tabIndex={isFocused ? 0 : -1}
    // biome-ignore lint/a11y/useSemanticElements: its okay
    role="button"
    onKeyDown={(e) => {
      if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        onClick();
      }
    }}
  >
    {label}
  </div>
);

const PopoverHeader = () => (
  <div className="flex w-full justify-between items-center px-1 py-1">
    <span className="text-gray-9 text-[13px]">Display Properties...</span>
    <KeyboardButton shortcut="D" />
  </div>
);

export const DisplayPopover = ({ children }: PropsWithChildren) => {
  const [open, setOpen] = useState(false);
  const [focusedIndex, setFocusedIndex] = useState<number | null>(null);
  const { displayProperties, toggleDisplayProperty } = useLogsContext();

  useKeyboardShortcut("d", () => {
    setOpen((prev) => !prev);
    if (!open) {
      setFocusedIndex(0);
    }
  });

  const handleKeyNavigation = useCallback(
    (e: KeyboardEvent) => {
      const itemsPerRow = Math.floor(384 / 120); // Approximate width / item width
      const totalItems = DISPLAY_PROPERTIES.length;
      const currentRow = Math.floor((focusedIndex ?? 0) / itemsPerRow);
      const currentCol = (focusedIndex ?? 0) % itemsPerRow;

      const moveToIndex = (newIndex: number) => {
        e.preventDefault();
        setFocusedIndex(Math.max(0, Math.min(newIndex, totalItems - 1)));
      };

      switch (e.key) {
        case "ArrowRight":
        case "l": {
          moveToIndex((focusedIndex ?? -1) + 1);
          break;
        }
        case "ArrowLeft":
        case "h": {
          moveToIndex((focusedIndex ?? 1) - 1);
          break;
        }
        case "ArrowDown":
        case "j": {
          const nextRowIndex = (currentRow + 1) * itemsPerRow + currentCol;
          if (nextRowIndex < totalItems) {
            moveToIndex(nextRowIndex);
          }
          break;
        }
        case "ArrowUp":
        case "k": {
          const prevRowIndex = (currentRow - 1) * itemsPerRow + currentCol;
          if (prevRowIndex >= 0) {
            moveToIndex(prevRowIndex);
          }
          break;
        }
        case "Enter":
        case " ": {
          if (focusedIndex !== null) {
            const prop = DISPLAY_PROPERTIES[focusedIndex];
            if (isDisplayProperty(prop.id)) {
              toggleDisplayProperty(prop.id);
            }
          }
          break;
        }
      }
    },
    [focusedIndex, toggleDisplayProperty],
  );

  useEffect(() => {
    if (!open) {
      setFocusedIndex(null);
    }
  }, [open]);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent
        className="bg-gray-1 dark:bg-black drop-shadow-2xl transform-gpu p-2 border-gray-6 rounded-lg w-96"
        align="start"
        onKeyDown={handleKeyNavigation}
      >
        <div className="flex flex-col gap-2">
          <PopoverHeader />
          <div className="flex flex-wrap gap-2">
            {DISPLAY_PROPERTIES.map((prop, index) => (
              <DisplayPropertyItem
                key={prop.id}
                label={prop.label}
                selected={displayProperties.has(prop.id)}
                onClick={() => {
                  if (isDisplayProperty(prop.id)) {
                    toggleDisplayProperty(prop.id);
                  }
                }}
                isFocused={focusedIndex === index}
                index={index}
              />
            ))}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
};
