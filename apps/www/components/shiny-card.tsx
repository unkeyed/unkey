"use client";

import { useMousePosition } from "@/lib/mouse";
import type React from "react";
import { type PropsWithChildren, useCallback, useEffect, useRef, useState } from "react";

type ShinyCardGroupProps = {
  children: React.ReactNode;
  className?: string;
  refresh?: boolean;
};

export const ShinyCardGroup: React.FC<ShinyCardGroupProps> = ({
  children,
  className = "",
  refresh = false,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const mousePosition = useMousePosition();
  const mouse = useRef<{ x: number; y: number }>({ x: 0, y: 0 });
  const containerSize = useRef<{ w: number; h: number }>({ w: 0, h: 0 });
  const [boxes, setBoxes] = useState<Array<HTMLElement>>([]);

  useEffect(() => {
    containerRef.current &&
      setBoxes(Array.from(containerRef.current.children).map((el) => el as HTMLElement));
  }, []);

  useEffect(() => {
    initContainer();
    window.addEventListener("resize", initContainer);

    return () => {
      window.removeEventListener("resize", initContainer);
    };
  }, []);

  const initContainer = useCallback(() => {
    if (containerRef.current) {
      containerSize.current.w = containerRef.current.offsetWidth;
      containerSize.current.h = containerRef.current.offsetHeight;
    }
  }, []);

  // biome-ignore lint: works fine
  const onMouseMove = useCallback(() => {
    if (containerRef.current) {
      const rect = containerRef.current.getBoundingClientRect();
      const { w, h } = containerSize.current;
      const x = mousePosition.x - rect.left;
      const y = mousePosition.y - rect.top;
      const inside = x < w && x > 0 && y < h && y > 0;
      if (inside) {
        mouse.current.x = x;
        mouse.current.y = y;
        boxes.forEach((box) => {
          const boxX = -(box.getBoundingClientRect().left - rect.left) + mouse.current.x;
          const boxY = -(box.getBoundingClientRect().top - rect.top) + mouse.current.y;
          box.style.setProperty("--mouse-x", `${boxX}px`);
          box.style.setProperty("--mouse-y", `${boxY}px`);
        });
      }
    }
  }, [mousePosition.x, mousePosition.y]);

  useEffect(() => {
    onMouseMove();
  }, [onMouseMove]);

  useEffect(() => {
    if (!refresh) {
      return;
    }
    initContainer();
  }, [initContainer, refresh]);

  return (
    <div className={className} ref={containerRef}>
      {children}
    </div>
  );
};

type ShinyCardProps = {
  children: React.ReactNode;
  className?: string;
  shine?: string;
};

export const ShinyCard: React.FC<PropsWithChildren<ShinyCardProps>> = ({
  children,
  className = "",
  shine = "#ffffff",
}) => {
  return (
    <div
      className={`relative bg-neutral-800 rounded-4xl p-px
    after:absolute after:inset-0 after:rounded-[inherit] after:opacity-0 after:transition-opacity after:duration-500 after:[background:_radial-gradient(250px_circle_at_var(--mouse-x)_var(--mouse-y),${shine},transparent)] after:group-hover:opacity-100 after:z-10 overflow-hidden ${className}`}
    >
      {children}
    </div>
  );
};

export const WhiteShinyCard: React.FC<PropsWithChildren<ShinyCardProps>> = ({
  children,
  className = "",
}) => {
  return (
    <div
      className={`relative bg-neutral-800 rounded-4xl p-px
    after:absolute after:inset-0 after:rounded-[inherit] after:opacity-0 after:transition-opacity after:duration-500 after:[background:_radial-gradient(250px_circle_at_var(--mouse-x)_var(--mouse-y),theme(colors.gray.500),transparent)] after:group-hover:opacity-100 after:z-10 overflow-hidden ${className}`}
    >
      {children}
    </div>
  );
};
