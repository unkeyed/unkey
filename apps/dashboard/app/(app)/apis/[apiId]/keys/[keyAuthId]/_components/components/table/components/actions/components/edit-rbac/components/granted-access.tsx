import { Page2 } from "@unkey/icons";
import { AnimatePresence, motion } from "framer-motion";
import { useEffect, useMemo, useState } from "react";

export const GrantedAccess = ({
  slugs = [],
  totalCount,
  isLoading,
}: {
  slugs?: string[];
  totalCount?: number;
  isLoading: boolean;
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
      <motion.div
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
        <Page2 size="sm-regular" className="text-grayA-11" />
        <span className="text-gray-11 text-xs font-mono">{slug}</span>
      </motion.div>
    ));
  }, [stableSlugs]);

  if (stableSlugs.length === 0 && !isLoading) {
    return null;
  }

  return (
    <motion.div
      layout
      className="space-y-3"
      initial={{ opacity: 0, height: 0 }}
      animate={{ opacity: 1, height: "auto" }}
      exit={{ opacity: 0, height: 0 }}
      transition={{ duration: 0.3, ease: "easeInOut" }}
    >
      <motion.div layout className="flex gap-2 items-center">
        <div className="font-medium text-sm text-gray-12">Granted Access</div>
        <motion.div
          key={totalCount}
          initial={{ scale: 0.8, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          transition={{ type: "spring", stiffness: 400, damping: 25 }}
          className={`
            rounded-full border bg-grayA-3 border-grayA-3 w-[22px] h-[18px] 
            flex items-center justify-center font-medium text-[11px] text-grayA-12
            ${isLoading ? "animate-pulse" : ""}
          `}
        >
          {isLoading ? "..." : totalCount}
        </motion.div>
      </motion.div>

      <motion.div
        layout
        className="h-[1px] bg-grayA-3 w-full"
        initial={{ scaleX: 0 }}
        animate={{ scaleX: 1 }}
        transition={{ duration: 0.4, delay: 0.1 }}
      />

      <motion.div
        layout
        className={`
          flex flex-wrap gap-1 items-center min-h-[2rem]
          transition-opacity duration-300 ease-in-out
          ${isLoading ? "opacity-50" : "opacity-100"}
        `}
      >
        <AnimatePresence mode="popLayout">
          {isLoading ? (
            <motion.div
              className="flex gap-1"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
            >
              {[1, 2, 3].map((i) => (
                <motion.div
                  key={i}
                  initial={{ opacity: 0, scale: 0.8 }}
                  animate={{ opacity: 1, scale: 1 }}
                  transition={{ delay: i * 0.1 }}
                  className="h-7 w-20 bg-grayA-4 rounded-md animate-pulse"
                />
              ))}
            </motion.div>
          ) : stableSlugs.length > 0 ? (
            memoizedSlugs
          ) : (
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              transition={{ duration: 0.3 }}
              className="text-grayA-9 text-xs italic py-2"
            >
              No permissions selected
            </motion.div>
          )}
        </AnimatePresence>
      </motion.div>
    </motion.div>
  );
};
