import { useEffect, useState } from "react";
// Adds bottom spacing to prevent the table from extending to the edge of the viewport
const BREATHING_SPACE = 20;
export const useTableHeight = (containerRef: React.RefObject<HTMLDivElement>) => {
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
