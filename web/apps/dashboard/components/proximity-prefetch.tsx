"use client";
import { useRouter } from "next/navigation";
import type { ReactNode } from "react";
import { useCallback, useEffect, useRef, useState } from "react";

type ProximityPrefetchProps = {
  /** The content to wrap and monitor for proximity */
  children: ReactNode;
  /** The route path to prefetch when mouse enters proximity. Optional if only using onEnterProximity */
  route?: string;
  /** Distance in pixels from element center to trigger prefetch. Default: 500px */
  distance?: number;
  /** Debounce delay in milliseconds for proximity checks. Default: 100ms */
  debounceDelay?: number;
  /** Callback fired once when mouse enters proximity threshold */
  onEnterProximity?: () => void | Promise<void>;
};

/**
 * Wraps children and prefetches a Next.js route when the user's mouse enters a defined proximity.
 *
 * Triggers once per route/element and resets when the route changes.
 * Useful for preloading data or routes before user interaction.
 *
 * @example
 * // Basic route prefetch
 * <ProximityPrefetch route="/projects/123">
 *   <ProjectCard />
 * </ProximityPrefetch>
 *
 * @example
 * // Custom callback for data preloading
 * <ProximityPrefetch
 *   route="/projects/123"
 *   onEnterProximity={async () => {
 *     await fetch('/api/projects/123/metrics');
 *   }}
 * >
 *   <ProjectCard />
 * </ProximityPrefetch>
 *
 * @example
 * // Adjust sensitivity
 * <ProximityPrefetch
 *   route="/projects/123"
 *   distance={300}
 *   debounceDelay={150}
 * >
 *   <ProjectCard />
 * </ProximityPrefetch>
 */
export function ProximityPrefetch({
  children,
  route,
  distance = 500,
  debounceDelay = 100,
  onEnterProximity,
}: ProximityPrefetchProps) {
  const router = useRouter();
  const containerRef = useRef<HTMLDivElement>(null);
  const [mousePosition, setMousePosition] = useState({ x: 0, y: 0 });
  const hasTriggered = useRef(false);
  const debounceTimeout = useRef<NodeJS.Timeout | undefined>(undefined);

  const checkProximity = useCallback(() => {
    // Skip if element doesn't exist or already triggered
    if (!containerRef.current || hasTriggered.current) {
      return;
    }

    // Calculate distance from mouse to element center
    const rect = containerRef.current.getBoundingClientRect();
    const centerX = rect.left + rect.width / 2;
    const centerY = rect.top + rect.height / 2;
    const distanceFromCenter = Math.sqrt(
      (mousePosition.x - centerX) ** 2 + (mousePosition.y - centerY) ** 2,
    );

    // Trigger prefetch and callback when mouse enters proximity threshold
    if (distanceFromCenter < distance) {
      hasTriggered.current = true;

      if (route) {
        router.prefetch(route);
      }

      if (onEnterProximity) {
        Promise.resolve(onEnterProximity()).catch((error) => {
          console.error("ProximityPrefetch callback error:", error);
        });
      }
    }
  }, [mousePosition, distance, route, router, onEnterProximity]);

  // Tracks the mouse position
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      setMousePosition({ x: e.clientX, y: e.clientY });
    };

    window.addEventListener("mousemove", handleMouseMove, { passive: true });
    return () => window.removeEventListener("mousemove", handleMouseMove);
  }, []);

  useEffect(() => {
    // Skip checking proximity until mouse has moved from initial (0,0) position
    if (mousePosition.x === 0 && mousePosition.y === 0) {
      return;
    }

    // Debounce proximity checks to avoid excessive calculations on every mouse move
    clearTimeout(debounceTimeout.current);
    debounceTimeout.current = setTimeout(checkProximity, debounceDelay);

    return () => clearTimeout(debounceTimeout.current);
  }, [mousePosition, checkProximity, debounceDelay]);

  useEffect(() => {
    hasTriggered.current = false;
  }, []);

  return <div ref={containerRef}>{children}</div>;
}
