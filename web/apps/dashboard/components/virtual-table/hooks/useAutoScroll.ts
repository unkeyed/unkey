import { useEffect, useRef } from "react";

type UseAutoScrollParams = {
  enabled: boolean;
  scrollRef: React.RefObject<HTMLDivElement>;
  anchorRef: React.RefObject<HTMLDivElement>;
  dataLength: number;
  isLoading: boolean;
  expandedCount: number;
};

const NEAR_BOTTOM_MARGIN_PX = 50;

export function useAutoScroll({
  enabled,
  scrollRef,
  anchorRef,
  dataLength,
  isLoading,
  expandedCount,
}: UseAutoScrollParams): void {
  const isNearBottom = useRef(true);
  const prevLength = useRef(dataLength);
  const rafId = useRef(0);

  useEffect(() => {
    if (!enabled) {
      return;
    }
    const container = scrollRef.current;
    const anchor = anchorRef.current;
    if (!container || !anchor) {
      return;
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        isNearBottom.current = entry.isIntersecting;
      },
      {
        root: container,
        rootMargin: `0px 0px ${NEAR_BOTTOM_MARGIN_PX}px 0px`,
        threshold: 0,
      },
    );

    observer.observe(anchor);
    return () => observer.disconnect();
  }, [enabled, scrollRef, anchorRef]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: dataLength drives scroll on new data
  useEffect(() => {
    if (dataLength === prevLength.current) {
      return;
    }
    prevLength.current = dataLength;

    if (!enabled || !isNearBottom.current || isLoading || expandedCount > 0) {
      return;
    }

    cancelAnimationFrame(rafId.current);
    rafId.current = requestAnimationFrame(() => {
      anchorRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
    });
    return () => cancelAnimationFrame(rafId.current);
  }, [enabled, dataLength, isLoading, expandedCount, anchorRef]);
}
