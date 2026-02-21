import { useEffect, useState } from "react";
import { BREATHING_SPACE } from "../constants/constants";

/**
 * Calculate dynamic table height based on viewport
 * Adds breathing space to prevent table from extending to viewport edge
 */
export const useTableHeight = (containerRef: React.RefObject<HTMLDivElement | null>) => {
  const [fixedHeight, setFixedHeight] = useState(0);

  useEffect(() => {
    const calculateHeight = () => {
      if (!containerRef.current) {
        return;
      }
      const rect = containerRef.current.getBoundingClientRect();
      const availableHeight = window.innerHeight - rect.top - BREATHING_SPACE;
      setFixedHeight(Math.max(availableHeight, 0));
    };

    calculateHeight();

    const resizeObserver = new ResizeObserver(calculateHeight);
    window.addEventListener("resize", calculateHeight);

    if (containerRef.current) {
      resizeObserver.observe(containerRef.current);
    }

    return () => {
      resizeObserver.disconnect();
      window.removeEventListener("resize", calculateHeight);
    };
  }, [containerRef]);

  return fixedHeight;
};
