"use client";

import { useEffect, useState } from "react";

export const scrollFadeClass =
  "data-[more-below]:[mask-image:linear-gradient(to_bottom,black_calc(100%-40px),transparent)]";

// Rounding and focus rings leave a few pixels of overflow on cards that fit.
const MIN_HIDDEN_PX = 8;

export function useScrollFade() {
  const [element, setElement] = useState<HTMLElement | null>(null);
  useEffect(() => {
    if (!element) {
      return;
    }
    const update = () => {
      const below = element.scrollHeight - element.scrollTop - element.clientHeight > MIN_HIDDEN_PX;
      element.toggleAttribute("data-more-below", below);
    };
    update();
    const resize = new ResizeObserver(update);
    resize.observe(element);
    for (const child of element.children) {
      resize.observe(child);
    }
    const mutations = new MutationObserver(update);
    mutations.observe(element, { childList: true, subtree: true });
    element.addEventListener("scroll", update, { passive: true });
    return () => {
      resize.disconnect();
      mutations.disconnect();
      element.removeEventListener("scroll", update);
    };
  }, [element]);
  return setElement;
}
