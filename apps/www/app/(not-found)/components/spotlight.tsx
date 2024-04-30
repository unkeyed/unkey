"use client";

import { cn } from "@/lib/utils";
import React from "react";

const CONTAINER_CN = "pointer-events-none absolute inset-0 overflow-hidden transition-opacity";
const SPOTLIGHT_WRAPPER_CN =
  "absolute aspect-square min-w-[200vw] min-h-[200vh] translate-x-[calc(var(--tx)-50%)] translate-y-[calc(var(--ty)-50%)]";
const SPOTLIGHT_CN = "absolute inset-0 bg-center bg-no-repeat";

export function Spotlight() {
  const containerRef = React.useRef<HTMLDivElement>(null);
  const areaRef = React.useRef<HTMLDivElement>(null);

  const [isTouch, setIsTouch] = React.useState(true);
  const [isMouseWithinArea, setIsMouseWithinArea] = React.useState(false);

  const maskVisible = !isTouch && isMouseWithinArea;

  React.useEffect(() => {
    const container = containerRef.current;
    const area = areaRef.current;
    if (!container || !area) {
      return;
    }

    // TODO: debounce
    const handleMouseMove = (e: MouseEvent) => {
      if ((e as any)?.sourceCapabilities?.firesTouchEvents) {
        setIsTouch(true);
        setIsMouseWithinArea(false);
        return;
      }

      const crect = container.getBoundingClientRect();
      const arect = area.getBoundingClientRect();
      const isWithinArea =
        e.clientX >= arect.left &&
        e.clientX <= arect.right &&
        e.clientY >= arect.top &&
        e.clientY <= arect.bottom;

      const x = e.clientX;
      const y = e.clientY;
      const cx = crect.left;
      const cy = crect.top;
      const offsetX = x - cx;
      const offsetY = y - cy;
      container.style.setProperty("--tx", `${offsetX}px`);
      container.style.setProperty("--ty", `${offsetY}px`);

      setIsTouch(false);
      setIsMouseWithinArea(isWithinArea);
    };

    function handleTouchStart() {
      setIsTouch(true);
    }

    window.addEventListener("touchstart", handleTouchStart);
    window.addEventListener("mousemove", handleMouseMove);

    return () => {
      window.removeEventListener("touchstart", handleTouchStart);
      window.removeEventListener("mousemove", handleMouseMove);
    };
  }, []);

  return (
    <div
      ref={containerRef}
      aria-hidden
      className={cn(
        CONTAINER_CN,
        "mix-blend-darken",
        maskVisible && "opacity-100 duration-500",
        !maskVisible && "opacity-0 duration-1000",
      )}
    >
      <div className={cn(SPOTLIGHT_WRAPPER_CN, "mix-blend-darken")}>
        <div
          className={cn(
            SPOTLIGHT_CN,
            "[background-image:radial-gradient(300px_300px_at_center,transparent,black_70%)]",
          )}
        />
      </div>

      {/* Usable area */}
      <div aria-hidden className="absolute inset-0 pointer-events-none container">
        <div ref={areaRef} className="w-full h-full" />
      </div>
    </div>
  );
}
