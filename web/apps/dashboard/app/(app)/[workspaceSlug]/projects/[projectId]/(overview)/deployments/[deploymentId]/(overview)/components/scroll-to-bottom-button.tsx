"use client";
import { ChevronDown } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import { useProjectData } from "../../../../data-provider";
import { useDeployment } from "../../layout-provider";

export function ScrollToBottomButton() {
  const { deploymentId } = useDeployment();
  const { getDeploymentById } = useProjectData();

  const deployment = getDeploymentById(deploymentId);
  const isVisible = deployment?.status !== "ready";

  const handleScrollToBottom = () => {
    const container = document.getElementById("deployment-scroll-container");
    if (container) {
      container.scrollTo({
        top: container.scrollHeight,
        behavior: "smooth",
      });
    }
  };

  return (
    <button
      type="button"
      onClick={handleScrollToBottom}
      className={cn(
        "fixed bottom-6 right-6 z-20",
        "flex items-center gap-2 px-4 py-2.5 rounded-full",
        "shadow-lg hover:shadow-xl hover:scale-105",
        "border border-grayA-4",
        "transition-all duration-300 ease-out",
        "overflow-hidden",
        "before:absolute before:inset-0 before:bg-gradient-to-r before:from-infoA-5 before:to-transparent before:-z-10",
        "after:absolute after:inset-0 after:bg-gray-1 after:dark:bg-black after:-z-20",
        isVisible ? "opacity-100 translate-y-0" : "opacity-0 translate-y-4 pointer-events-none",
      )}
      aria-label="Scroll to bottom of build logs"
    >
      {/* Shimmer animation overlay */}
      <div
        className="absolute inset-0 bg-gradient-to-r from-transparent via-white/40 to-transparent w-[150%]"
        style={{
          animation: "shimmer 1.2s ease-in-out infinite",
        }}
      />
      <span className="text-xs text-infoA-11 font-medium relative z-10">Building...</span>
      <ChevronDown iconSize="sm-regular" className="text-info-11 relative z-10" />
      <style jsx>{`
        @keyframes shimmer {
          0% {
            transform: translateX(-100%);
          }
          100% {
            transform: translateX(100%);
          }
        }
      `}</style>
    </button>
  );
}
