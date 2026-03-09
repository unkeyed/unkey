import { useEffect, useRef } from "react";

type UseAutoScrollParams = {
  enabled: boolean;
  parentRef: React.RefObject<HTMLDivElement>;
  data: unknown[];
  isLoading: boolean;
  expandedCount: number;
};

const SCROLL_THRESHOLD_PX = 10;

export function useAutoScroll({
  enabled,
  parentRef,
  data,
  isLoading,
  expandedCount,
}: UseAutoScrollParams): void {
  const userScrolledUp = useRef(false);

  useEffect(() => {
    if (!enabled) {
      return;
    }
    const el = parentRef.current;
    if (!el) {
      return;
    }
    const onScroll = () => {
      userScrolledUp.current =
        el.scrollHeight - el.scrollTop - el.clientHeight > SCROLL_THRESHOLD_PX;
    };
    el.addEventListener("scroll", onScroll);
    return () => el.removeEventListener("scroll", onScroll);
  }, [enabled, parentRef]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: data is intentionally listed to trigger scroll on new data
  useEffect(() => {
    if (!enabled || userScrolledUp.current || isLoading || expandedCount > 0) {
      return;
    }
    parentRef.current?.scrollTo({ top: parentRef.current.scrollHeight, behavior: "smooth" });
  }, [enabled, data, isLoading, expandedCount, parentRef]);
}
