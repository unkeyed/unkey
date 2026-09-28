"use client";
import { useEffect, useRef, useState } from "react";

export function useNearViewport<T extends Element>(rootMargin = "300px") {
  const ref = useRef<T>(null);
  const [isNear, setIsNear] = useState(false);

  useEffect(() => {
    const element = ref.current;
    if (!element || isNear) {
      return;
    }
    if (typeof IntersectionObserver === "undefined") {
      setIsNear(true);
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          setIsNear(true);
          observer.disconnect();
        }
      },
      { rootMargin },
    );
    observer.observe(element);
    return () => observer.disconnect();
  }, [isNear, rootMargin]);

  return { ref, isNear };
}
