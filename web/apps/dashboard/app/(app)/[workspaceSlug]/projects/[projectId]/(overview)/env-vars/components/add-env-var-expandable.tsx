import { DoubleChevronRight } from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useEffect, useRef } from "react";

type AddEnvVarExpandableProps = {
  tableDistanceToTop: number;
  isOpen: boolean;
  onClose: () => void;
};

export const AddEnvVarExpandable = ({
  tableDistanceToTop,
  isOpen,
  onClose,
}: AddEnvVarExpandableProps) => {
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    const handleClickOutside = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        onClose();
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [isOpen, onClose]);

  return (
    <div className="flex">
      <div
        ref={panelRef}
        className={cn(
          "fixed right-3 bg-gray-1 border border-grayA-4 rounded-xl w-150 overflow-hidden z-50 pb-8",
          "transition-all duration-300 ease-out",
          "shadow-md",
          isOpen ? "translate-x-0 opacity-100" : "translate-x-full opacity-0",
        )}
        style={{
          top: `${tableDistanceToTop + 12}px`,
          height: `calc(100vh - ${tableDistanceToTop + 24}px)`,
          willChange: isOpen ? "transform, opacity" : "auto",
        }}
      >
        <div className="h-full overflow-y-auto">
          {/* Header */}
          <div className="flex items-start justify-between border-b border-grayA-4 px-5 py-4 sticky top-0 z-10">
            <div className="flex flex-col">
              <span className="text-gray-12 font-medium text-base leading-8">
                Add Environment Variable
              </span>
              <span className="text-gray-9 text-[13px] leading-5">
                Set a key-value pair for your project.
              </span>
            </div>
            <InfoTooltip
              content="Close"
              asChild
              position={{
                side: "bottom",
                align: "end",
              }}
            >
              <Button variant="ghost" size="icon" onClick={onClose} className="mt-0.5">
                <DoubleChevronRight
                  iconSize="lg-medium"
                  className="text-gray-8 transition-transform duration-300 ease-out group-hover:text-gray-12"
                />
              </Button>
            </InfoTooltip>
          </div>

          {/* Animated content with stagger effect */}
          <div
            className={cn(
              "transition-all duration-500 ease-out",
              isOpen ? "translate-x-0 opacity-100" : "translate-x-6 opacity-0",
            )}
            style={{
              transitionDelay: isOpen ? "150ms" : "0ms",
            }}
          >
            <div className="p-4" />
          </div>
        </div>
      </div>
    </div>
  );
};
