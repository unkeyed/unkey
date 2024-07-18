import { AnimatePresence, motion, useInView, useWillChange } from "framer-motion";
import React, { type ReactElement, useEffect, useMemo, useState, useRef } from "react";

export const AnimatedList = React.memo(
  ({
    className,
    children,
    delay = 1000,
  }: {
    className?: string;
    children: React.ReactNode;
    delay?: number;
  }) => {
    const [index, setIndex] = useState(0);

    const childrenArray = React.Children.toArray(children);

    const ref = useRef(null);
    const inView = useInView(ref);
    const timeoutRef = useRef<NodeJS.Timeout>();

    useEffect(() => {
      return () => {
        if (timeoutRef.current) {
          clearTimeout(timeoutRef.current);
        }
      };
    }, []);

    useEffect(() => {
      if (inView) {
        const interval = setInterval(() => {
          setIndex((prevIndex) => {
            if (prevIndex >= childrenArray.length - 1) {
              timeoutRef.current = setTimeout(() => {
                setIndex(-1);
              }, 1000);
            }
            return prevIndex + 1;
          });
        }, delay);

        return () => clearInterval(interval);
      }
    }, [childrenArray.length, delay, inView]);

    const itemsToShow = useMemo(() => {
      if (index === -1) {
        timeoutRef.current = setTimeout(() => {
          setIndex(0);
        }, 1000);
        return [];
      }
      return childrenArray.slice(0, index + 1).reverse();
    }, [index, childrenArray]);

    return (
      <div className={`flex flex-col items-center gap-4 ${className}`} ref={ref}>
        <AnimatePresence>
          {itemsToShow.map((item) => (
            <AnimatedListItem key={(item as ReactElement).key}>{item}</AnimatedListItem>
          ))}
        </AnimatePresence>
      </div>
    );
  },
);

AnimatedList.displayName = "AnimatedList";

export function AnimatedListItem({ children }: { children: React.ReactNode }) {
  const willChange = useWillChange();

  const animations = {
    initial: { scale: 0, opacity: 0 },
    whileInView: { opacity: 1 },
    viewport: { once: true, amount: 0.5 },
    animate: { scale: 1, opacity: 1, originY: 0 },
    exit: { scale: 0, opacity: 0 },
    transition: { type: "spring", stiffness: 350, damping: 40 },
  };
  return (
    <motion.div {...animations} layout className="mx-auto" style={{ willChange }}>
      {children}
    </motion.div>
  );
}
