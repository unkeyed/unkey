import { ErrorBoundary } from "@/components/error-boundary";
import { ArrowDottedRotateAnticlockwise, TriangleWarning } from "@unkey/icons";
import { Button } from "@unkey/ui";
import type { PropsWithChildren } from "react";
import { TreeElementNode } from "../tree/tree-element-node";

export const CanvasBoundary = ({ children }: PropsWithChildren) => {
  return (
    <ErrorBoundary
      fallback={(error, reset) => (
        <TreeElementNode id="error-state" position={{ x: 0, y: 250 }}>
          <div className="w-[400px]">
            {/* Background card with glow */}
            <div
              className="relative dark:bg-grayA-1 bg-white border border-grayA-5 rounded-xl overflow-hidden"
              style={{
                boxShadow:
                  "0 4px 16px -4px rgba(0,0,0,0.15), 0 0 0 1px hsl(var(--errorA-3)), 0 0 30px color-mix(in srgb, hsl(var(--errorA-9)) 20%, transparent)",
              }}
            >
              {/* Error state indicator - top section */}
              <div className="h-12 border-b border-grayA-4 flex items-center px-4 gap-3">
                {/* Icon container */}
                <div className="size-6 rounded-md bg-redA-3 border border-grayA-4 flex items-center justify-center flex-shrink-0">
                  <TriangleWarning className="text-red-11" iconSize="md-medium" />
                </div>
                {/* Title */}
                <span className="text-sm font-medium text-gray-12">
                  Failed to render network tree
                </span>
                {/* Status dot */}
                <div className="ml-auto relative size-[10px]">
                  <div
                    className="absolute inset-0 rounded-full bg-redA-3 opacity-60"
                    style={{
                      animation: "breathe-ring 2s ease-in-out infinite",
                    }}
                  />
                  <div className="absolute inset-0 rounded-full bg-red-8" />
                </div>
              </div>

              {/* Error message */}
              <div className="p-4 space-y-2">
                <div className="text-xs font-medium text-gray-11">{error.message}</div>
                <div className="text-xs text-gray-9">Check console for details</div>
              </div>

              {/* Retry button */}
              <div className="p-4 pt-0">
                <Button
                  onClick={reset}
                  variant="outline"
                  className="w-full h-8 px-3 bg-grayA-3 hover:bg-grayA-4 border border-grayA-5 text-gray-12 text-xs font-medium rounded-lg transition-all duration-200 ease-out hover:ring-1 hover:ring-gray-7 flex items-center justify-center gap-2"
                >
                  <ArrowDottedRotateAnticlockwise />
                  Retry Layout
                </Button>
              </div>
            </div>

            <style>{`
                    @keyframes breathe-ring {
                      0%, 100% {
                        transform: scale(1);
                        opacity: 0.6;
                      }
                      50% {
                        transform: scale(2);
                        opacity: 0;
                      }
                    }
                  `}</style>
          </div>
        </TreeElementNode>
      )}
    >
      {children}
    </ErrorBoundary>
  );
};
