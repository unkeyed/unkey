import { type RefObject, useEffect, useState } from "react";

/**
 * Tracks whether the pointer is over the referenced element. Used to control
 * Radix Popover / similar primitives with a hover trigger.
 *
 * Based on https://gist.github.com/MurkyMeow/2d0f3cdd1a9034dcc1d9b8348799a6e2
 */
export function useHover<T extends HTMLElement>(
  elementRef: RefObject<T | null>,
): boolean {
  const [value, setValue] = useState(false);

  useEffect(() => {
    const element = elementRef.current;
    if (!element) return;

    const enter = () => setValue(true);
    const leave = () => setValue(false);

    element.addEventListener("mouseenter", enter);
    element.addEventListener("mouseleave", leave);
    document.addEventListener("mouseleave", leave);

    return () => {
      element.removeEventListener("mouseenter", enter);
      element.removeEventListener("mouseleave", leave);
      document.removeEventListener("mouseleave", leave);
    };
  }, [elementRef]);

  return value;
}
