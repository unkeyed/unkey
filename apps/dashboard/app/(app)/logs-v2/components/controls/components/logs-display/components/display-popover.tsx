import {
  isDisplayProperty,
  useLogsContext,
} from "@/app/(app)/logs-v2/context/logs";
import { useKeyboardShortcut } from "@/app/(app)/logs-v2/hooks/use-keyboard-shortcut";
import { KeyboardButton } from "@/components/keyboard-button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { type PropsWithChildren, useState } from "react";

const DISPLAY_PROPERTIES = [
  { id: "time", label: "Time" },
  { id: "response_status", label: "Status" },
  { id: "method", label: "Method" },
  { id: "path", label: "Path" },
  { id: "response_body", label: "Response Body" },
  { id: "request_id", label: "Request ID" },
  { id: "workspace_id", label: "Workspace ID" },
  { id: "host", label: "Host" },
  { id: "request_headers", label: "Request Headers" },
  { id: "request_body", label: "Request Body" },
  { id: "response_headers", label: "Response Headers" },
];

const DisplayPropertyItem = ({
  label,
  selected,
  onClick,
}: {
  label: string;
  selected: boolean;
  onClick: () => void;
}) => (
  <div
    className={`font-medium text-xs p-1.5 rounded-md hover:bg-gray-4 cursor-pointer whitespace-nowrap
      ${selected ? "bg-gray-4 text-gray-12" : "text-gray-9"}`}
    onClick={onClick}
  >
    {label}
  </div>
);

const PopoverHeader = () => {
  return (
    <div className="flex w-full justify-between items-center px-1 py-1">
      <span className="text-gray-9 text-[13px]">Display Properties...</span>
      <KeyboardButton shortcut="D" />
    </div>
  );
};

export const DisplayPopover = ({ children }: PropsWithChildren) => {
  const [open, setOpen] = useState(false);
  const { displayProperties, toggleDisplayProperty } = useLogsContext();

  useKeyboardShortcut("d", () => {
    setOpen((prev) => !prev);
  });

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent
        className="bg-gray-1 dark:bg-black drop-shadow-2xl p-2 border-gray-6 rounded-lg w-96"
        align="start"
      >
        <div className="flex flex-col gap-2">
          <PopoverHeader />
          <div className="flex flex-wrap gap-2">
            {DISPLAY_PROPERTIES.map((prop) => (
              <DisplayPropertyItem
                key={prop.id}
                label={prop.label}
                selected={displayProperties.has(prop.id as any)}
                onClick={() => {
                  if (isDisplayProperty(prop.id)) {
                    toggleDisplayProperty(prop.id);
                  }
                }}
              />
            ))}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
};

export default DisplayPopover;
