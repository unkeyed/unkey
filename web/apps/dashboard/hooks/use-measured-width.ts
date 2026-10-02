"use client";

import { useCallback, useState } from "react";

export function useMeasuredWidth<T extends HTMLElement>() {
  const [width, setWidth] = useState<number>();

  const measureRef = useCallback((el: T | null) => {
    if (!el) {
      return;
    }
    const observer = new ResizeObserver((entries) => {
      const measured = entries[0]?.contentRect.width;
      if (measured !== undefined) {
        setWidth(measured);
      }
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  return { width, measureRef };
}
