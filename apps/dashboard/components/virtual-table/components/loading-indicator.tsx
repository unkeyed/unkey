import { cn } from "@/lib/utils";
import { Button } from "@unkey/ui";
import { useEffect, useRef, useState } from "react";

interface LoadMoreFooterProps {
  onLoadMore?: () => void;
  isFetchingNextPage?: boolean;
  totalVisible: number;
  totalCount: number;
  className?: string;
}

const PROXIMITY_THRESHOLD = 500;
export const LoadMoreFooter = ({
  onLoadMore,
  isFetchingNextPage = false,
  totalVisible,
  totalCount,
  className,
}: LoadMoreFooterProps) => {
  const [isFooterVisible, setIsFooterVisible] = useState(false);
  const [mousePosition, setMousePosition] = useState({ x: 0, y: 0 });
  const footerRef = useRef<HTMLDivElement>(null);

  // Make footer visible during fetching
  useEffect(() => {
    if (isFetchingNextPage) {
      setIsFooterVisible(true);
    }
  }, [isFetchingNextPage]);

  // Track mouse position
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      setMousePosition({ x: e.clientX, y: e.clientY });
    };

    window.addEventListener("mousemove", handleMouseMove);
    return () => {
      window.removeEventListener("mousemove", handleMouseMove);
    };
  }, []);

  // Check proximity
  useEffect(() => {
    if (footerRef.current && !isFetchingNextPage) {
      const footerRect = footerRef.current.getBoundingClientRect();

      // Calculate vertical distance to the footer
      const verticalDistance = Math.max(
        0,
        mousePosition.y < footerRect.top
          ? footerRect.top - mousePosition.y
          : mousePosition.y > footerRect.bottom
            ? mousePosition.y - footerRect.bottom
            : 0,
      );

      // Calculate horizontal distance (only if cursor is outside the footer's width)
      const horizontalDistance =
        mousePosition.x < footerRect.left
          ? footerRect.left - mousePosition.x
          : mousePosition.x > footerRect.right
            ? mousePosition.x - footerRect.right
            : 0;

      // Use weighted distance (prioritize vertical proximity)
      const verticalWeight = 1;
      const horizontalWeight = 0.5;
      const weightedDistance =
        verticalDistance * verticalWeight + horizontalDistance * horizontalWeight;

      // Set visibility based on weighted proximity
      if (weightedDistance < PROXIMITY_THRESHOLD) {
        const opacity = 1 - weightedDistance / PROXIMITY_THRESHOLD;
        setIsFooterVisible(opacity > 0.3);
      } else {
        setIsFooterVisible(false);
      }
    }
  }, [mousePosition, isFetchingNextPage]);

  if (!onLoadMore) {
    return null;
  }

  return (
    <div
      ref={footerRef}
      className={cn(
        "sticky bottom-0 left-0 right-0 w-full items-center justify-center flex",
        className,
      )}
    >
      <div
        className={cn(
          "w-[740px] border bg-gray-1 dark:bg-black border-gray-6 h-[60px]",
          "flex items-center justify-center p-[18px] rounded-[10px]",
          "drop-shadow-lg shadow-sm mb-5 transition-opacity duration-300",
          isFooterVisible || isFetchingNextPage ? "opacity-100" : "opacity-50",
        )}
      >
        <div className="flex w-full justify-between items-center text-[13px] text-accent-9">
          <div className="flex gap-2">
            <span>Viewing</span> <span className="text-accent-12">{totalVisible}</span>
            <span>of</span>
            <span className="text-grayA-12">{totalCount} </span>
            <span>keys</span>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={onLoadMore}
            loading={isFetchingNextPage}
            disabled={isFetchingNextPage}
          >
            Load more
          </Button>
        </div>
      </div>
    </div>
  );
};
