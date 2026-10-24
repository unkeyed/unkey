import { MotionProvider } from "@/components/motion-provider";
import { IconPage2Outline12 } from "@unkey/icons";
import { AnimatePresence, m } from "framer-motion";
import { useEffect, useMemo, useState } from "react";

const NO_SLUGS: string[] = [];

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

  const memoizedSlugs = useMemo(() => {
    return stableSlugs.map((slug, index) => (
      <m.div
        key={slug}
        layout
        initial={{ opacity: 0, scale: 0.8, y: 10 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        exit={{ opacity: 0, scale: 0.8, y: -10 }}
        transition={{
          type: "spring",
          stiffness: 500,
          damping: 30,
          mass: 0.8,
          delay: index * 0.02,
        }}
        className="flex gap-2 items-center bg-grayA-3 rounded-md p-1.5"
      >
        <IconPage2Outline12 className="text-grayA-11" />
        <span className="text-gray-11 text-xs font-mono">{slug}</span>
      </m.div>
    ));
  }, [stableSlugs]);

  return (
    <MotionProvider>
      <m.div
        layout
        className="flex flex-col gap-3"
        initial={{ opacity: 0, height: 0 }}
        animate={{ opacity: 1, height: "auto" }}
        exit={{ opacity: 0, height: 0 }}
        transition={{ duration: 0.3, ease: "easeInOut" }}
      >
        <m.div layout className="flex gap-2 items-center">
          <div className="font-medium text-sm text-gray-12">Granted Access</div>
          <m.div
            key={totalCount}
            initial={{ scale: 0.8, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            transition={{ type: "spring", stiffness: 400, damping: 25 }}
            className={`
            rounded-full border bg-grayA-3 border-grayA-3 w-[22px] h-[18px]
            flex items-center justify-center font-medium text-2xs text-grayA-12
            ${isLoading ? "animate-pulse" : ""}
          `}
          >
            {isLoading ? "..." : totalCount}
          </m.div>
        </m.div>

        <m.div
          layout
          className="h-px bg-grayA-3 w-full"
          initial={{ scaleX: 0 }}
          animate={{ scaleX: 1 }}
          transition={{ duration: 0.4, delay: 0.1 }}
        />

        <m.div
          layout
          className={`
          flex flex-wrap gap-1 items-center min-h-8
          transition-opacity duration-300 ease-in-out
          ${isLoading ? "opacity-50" : "opacity-100"}
        `}
        >
          <AnimatePresence mode="popLayout">
            {isLoading ? (
              <m.div
                className="flex gap-1"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
              >
                {[1, 2, 3].map((i) => (
                  <m.div
                    key={i}
                    initial={{ opacity: 0, scale: 0.8 }}
                    animate={{ opacity: 1, scale: 1 }}
                    transition={{ delay: i * 0.1 }}
                    className="h-7 w-20 bg-grayA-4 rounded-md animate-pulse"
                  />
                ))}
              </m.div>
            ) : hasError ? (
              <m.div
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -10 }}
                transition={{ duration: 0.3 }}
                className="text-error-11 text-xs py-2"
              >
                Could not load permissions for the selected roles
              </m.div>
            ) : stableSlugs.length > 0 ? (
              memoizedSlugs
            ) : (
              <m.div
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -10 }}
                transition={{ duration: 0.3 }}
                className="text-grayA-9 text-xs italic py-2"
              >
                No permissions selected
              </m.div>
            )}
          </AnimatePresence>
        </m.div>
      </m.div>
    </MotionProvider>
  );
};
