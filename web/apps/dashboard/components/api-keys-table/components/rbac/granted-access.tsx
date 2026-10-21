import { SPRING_POP, useLayoutMotion } from "@/hooks/use-layout-motion";
import { IconPage2Outline12 } from "@unkey/icons";
import { cn } from "cn";
import { useEffect, useMemo, useRef, useState } from "react";

const NO_SLUGS: string[] = [];

const CHIP_TIMING = { duration: 400, easing: SPRING_POP };
const MESSAGE_TIMING = { duration: 300, easing: "ease-out" };
const MESSAGE_KEYS = new Set(["loading", "error", "empty"]);

export const GrantedAccess = ({
  slugs = NO_SLUGS,
  totalCount,
  isLoading,
  hasError,
}: {
  slugs?: string[];
  totalCount?: number;
  isLoading: boolean;
  hasError: boolean;
}) => {
  const [stableSlugs, setStableSlugs] = useState<string[]>([]);
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isLoading && slugs) {
      setStableSlugs((prev) => {
        const newSlugsSet = new Set(slugs);
        const prevSlugsSet = new Set(prev);

        const retained = prev.filter((slug) => newSlugsSet.has(slug));

        const newOnes = slugs.filter((slug) => !prevSlugsSet.has(slug));

        return [...retained, ...newOnes];
      });
    }
  }, [slugs, isLoading]);

  useLayoutMotion(listRef, {
    move: CHIP_TIMING,
    enter: (key) =>
      MESSAGE_KEYS.has(key)
        ? key === "loading"
          ? { keyframes: [{ opacity: 0 }, { opacity: 1 }], timing: MESSAGE_TIMING }
          : {
              keyframes: [
                { opacity: 0, translate: "0 10px" },
                { opacity: 1, translate: "0 0" },
              ],
              timing: MESSAGE_TIMING,
            }
        : {
            keyframes: [
              { opacity: 0, scale: "0.8", translate: "0 10px" },
              { opacity: 1, scale: "1", translate: "0 0" },
            ],
            timing: CHIP_TIMING,
            staggerMs: 20,
          },
    exit: (key) =>
      MESSAGE_KEYS.has(key)
        ? {
            keyframes: [
              { opacity: 1, translate: "0 0" },
              { opacity: 0, translate: key === "loading" ? "0 0" : "0 -10px" },
            ],
            timing: MESSAGE_TIMING,
          }
        : {
            keyframes: [
              { opacity: 1, scale: "1", translate: "0 0" },
              { opacity: 0, scale: "0.8", translate: "0 -10px" },
            ],
            timing: CHIP_TIMING,
          },
  });

  const memoizedSlugs = useMemo(() => {
    return stableSlugs.map((slug) => (
      <div
        key={slug}
        data-motion-key={slug}
        className="flex gap-2 items-center bg-grayA-3 rounded-md p-1.5"
      >
        <IconPage2Outline12 className="text-grayA-11" />
        <span className="text-gray-11 text-xs font-mono">{slug}</span>
      </div>
    ));
  }, [stableSlugs]);

  return (
    <div className="grid grid-rows-[1fr] opacity-100 transition-[grid-template-rows,opacity] duration-300 ease-[ease-in-out] starting:grid-rows-[0fr] starting:opacity-0">
      <div className="flex min-h-0 flex-col gap-3 overflow-hidden">
        <div className="flex gap-2 items-center">
          <div className="font-medium text-sm text-gray-12">Granted Access</div>
          <div
            key={totalCount}
            className={cn(
              "rounded-full border bg-grayA-3 border-grayA-3 w-[22px] h-[18px] flex items-center justify-center font-medium text-2xs text-grayA-12",
              isLoading ? "animate-pulse" : "animate-badge-pop",
            )}
          >
            {isLoading ? "..." : totalCount}
          </div>
        </div>

        <div className="h-px bg-grayA-3 w-full transition-transform delay-100 duration-400 ease-[ease-out] starting:scale-x-0" />

        <div
          ref={listRef}
          className={cn(
            "relative flex flex-wrap gap-1 items-center min-h-8 transition-opacity duration-300 ease-in-out",
            isLoading ? "opacity-50" : "opacity-100",
          )}
        >
          {isLoading ? (
            <div key="loading" data-motion-key="loading" className="flex gap-1">
              {[1, 2, 3].map((i) => (
                <div
                  key={i}
                  style={{ transitionDelay: `${i * 100}ms`, transitionTimingFunction: SPRING_POP }}
                  className="h-7 w-20 bg-grayA-4 rounded-md animate-pulse transition-[opacity,scale] duration-400 starting:opacity-0 starting:scale-[0.8]"
                />
              ))}
            </div>
          ) : hasError ? (
            <div key="error" data-motion-key="error" className="text-error-11 text-xs py-2">
              Could not load permissions for the selected roles
            </div>
          ) : stableSlugs.length > 0 ? (
            memoizedSlugs
          ) : (
            <div key="empty" data-motion-key="empty" className="text-grayA-9 text-xs italic py-2">
              No permissions selected
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
