"use client";

import { type RefObject, useLayoutEffect, useRef } from "react";

// Spring curves sampled into CSS linear() easings so the browser plays them without a library.
// Stiffness 500, damping 30, mass 0.8 over 400ms.
export const SPRING_POP =
  "linear(0, 0.07, 0.226, 0.408, 0.581, 0.727, 0.839, 0.92, 0.974, 1.006, 1.022, 1.028, 1.028, 1.024, 1.019, 1.014, 1.009, 1.006, 1.003, 1.001, 1, 1, 0.999, 0.999, 1)";
// Critically damped, settling in about 290ms.
export const SPRING_SETTLE =
  "linear(0, 0.067, 0.206, 0.359, 0.501, 0.62, 0.717, 0.792, 0.849, 0.891, 0.922, 0.945, 0.961, 0.973, 0.981, 0.987, 0.991, 0.994, 0.996, 0.997, 1)";

type MotionStep = {
  keyframes: Keyframe[];
  timing: KeyframeAnimationOptions;
};

export type LayoutMotion = {
  enabled?: boolean;
  /** Play `enter` for the children present when the container mounts. */
  appear?: boolean;
  /** When set, children only animate on commits where this value changed. */
  dependency?: unknown;
  move?: KeyframeAnimationOptions;
  enter?: (key: string) => (MotionStep & { staggerMs?: number }) | undefined;
  exit?: (key: string) => MotionStep | undefined;
};

type Placement = {
  node: HTMLElement;
  left: number;
  top: number;
  width: number;
  height: number;
};

/**
 * Animates descendants of `container` marked with `data-motion-key`, the way framer-motion
 * `layout` with `AnimatePresence mode="popLayout"` does: a child that changed position slides
 * from its old spot, a new child plays `enter`, and a removed child plays `exit` in its old
 * spot without taking up space. The container must be positioned.
 */
export function useLayoutMotion(container: RefObject<HTMLElement | null>, motion: LayoutMotion) {
  const placements = useRef<{
    root: HTMLElement;
    byKey: Map<string, Placement>;
    dependency: unknown;
  } | null>(null);
  const latest = useRef(motion);
  latest.current = motion;

  useLayoutEffect(() => {
    const root = container.current;
    if (!root) {
      return;
    }
    const { enabled = true, appear = true, dependency, move, enter, exit } = latest.current;
    const nodes = [...root.querySelectorAll<HTMLElement>("[data-motion-key]")];
    const next = new Map<string, Placement>();
    for (const node of nodes) {
      const key = node.dataset.motionKey;
      if (key) {
        next.set(key, { node, ...offsetWithin(node, root) });
      }
    }
    const last = placements.current;
    const sameRoot = last?.root === root;
    const previous = sameRoot ? last.byKey : appear ? new Map<string, Placement>() : null;
    placements.current = { root, byKey: next, dependency };
    const unchanged =
      sameRoot && dependency !== undefined && Object.is(last.dependency, dependency);
    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (!previous || !enabled || unchanged || reduced) {
      return;
    }

    nodes.forEach((node, index) => {
      const key = node.dataset.motionKey;
      const now = key ? next.get(key) : undefined;
      if (!key || !now) {
        return;
      }
      const before = previous.get(key);
      if (!before) {
        const step = enter?.(key);
        if (step) {
          node.animate(step.keyframes, {
            ...step.timing,
            delay: (step.staggerMs ?? 0) * index,
            fill: "backwards",
          });
        }
        return;
      }
      const dx = before.left - now.left;
      const dy = before.top - now.top;
      if (move && (Math.abs(dx) > 0.5 || Math.abs(dy) > 0.5)) {
        node.animate([{ translate: `${dx}px ${dy}px` }, { translate: "0px 0px" }], move);
      }
    });

    for (const [key, before] of previous) {
      const step = exit?.(key);
      if (next.has(key) || !step || before.node.isConnected) {
        continue;
      }
      const ghost = before.node;
      delete ghost.dataset.motionKey;
      ghost.inert = true;
      ghost.setAttribute("aria-hidden", "true");
      Object.assign(ghost.style, {
        position: "absolute",
        left: `${before.left}px`,
        top: `${before.top}px`,
        width: `${before.width}px`,
        height: `${before.height}px`,
        margin: "0",
        pointerEvents: "none",
      });
      root.append(ghost);
      const removeGhost = () => ghost.remove();
      ghost
        .animate(step.keyframes, { ...step.timing, fill: "forwards" })
        .finished.then(removeGhost, removeGhost);
    }
  });
}

function offsetWithin(node: HTMLElement, root: HTMLElement) {
  let left = 0;
  let top = 0;
  let current: HTMLElement | null = node;
  while (current && current !== root) {
    left += current.offsetLeft;
    top += current.offsetTop;
    current = current.offsetParent instanceof HTMLElement ? current.offsetParent : null;
  }
  return { left, top, width: node.offsetWidth, height: node.offsetHeight };
}
