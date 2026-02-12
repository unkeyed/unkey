import { Book2, DoubleChevronRight } from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { ProjectDetailsContent } from "./project-details-content";

type ProjectDetailsExpandableProps = {
  tableDistanceToTop: number;
  isOpen: boolean;
  onClose: () => void;
};

export const ProjectDetailsExpandable = ({
  tableDistanceToTop,
  isOpen,
  onClose,
}: ProjectDetailsExpandableProps) => {
  return (
    <div className="flex">
      <div
        className={cn(
          "fixed right-0 bg-gray-1 border-l border-grayA-4 w-[360px] overflow-hidden z-50 pb-8",
          "transition-all duration-300 ease-out",
          "shadow-md",
          isOpen ? "translate-x-0 opacity-100" : "translate-x-full opacity-0",
        )}
        style={{
          top: `${tableDistanceToTop}px`,
          height: `calc(100vh - ${tableDistanceToTop}px)`,
          willChange: isOpen ? "transform, opacity" : "auto",
        }}
      >
        {/* Scrollable content container */}
        <div className="h-full overflow-y-auto">
          {/* Details Header */}
          <div className="h-10 flex items-center justify-between border-b border-grayA-4 px-4 bg-gray-1 sticky top-0 z-10">
            <div className="items-center flex gap-2.5 pl-0.5 py-2">
              <Book2 iconSize="md-medium" />
              <span className="text-accent-12 font-medium text-sm">Details</span>
            </div>
            <InfoTooltip
              content="Hide details"
              asChild
              position={{
                side: "bottom",
                align: "end",
              }}
            >
              <Button variant="ghost" size="icon" onClick={onClose}>
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
            <ProjectDetailsContent />
          </div>
        </div>
      </div>
    </div>
  );
};
