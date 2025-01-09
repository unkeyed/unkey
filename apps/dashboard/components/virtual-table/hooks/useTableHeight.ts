import { useEffect, useState } from "react";

export const useTableHeight = (
  containerRef: React.RefObject<HTMLDivElement>,
  headerHeight: number,
  tableBorder: number,
) => {
  const [fixedHeight, setFixedHeight] = useState(0);

  useEffect(() => {
    const calculateHeight = () => {
      if (!containerRef.current) {
        return;
      }
      const rect = containerRef.current.getBoundingClientRect();
      const totalHeaderHeight = headerHeight + tableBorder;
      const availableHeight = window.innerHeight - rect.top - totalHeaderHeight;
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
  }, [containerRef, headerHeight, tableBorder]);

  return fixedHeight;
};
