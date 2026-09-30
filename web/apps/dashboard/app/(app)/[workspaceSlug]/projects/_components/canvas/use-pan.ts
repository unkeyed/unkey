"use client";

import {
  type DragEvent,
  type KeyboardEvent,
  type MouseEvent,
  type PointerEvent,
  type RefObject,
  type UIEvent,
  useLayoutEffect,
  useRef,
  useState,
} from "react";

type Point = { x: number; y: number };

type Drag =
  | { phase: "idle" }
  | { phase: "pressing" | "panning"; pointerId: number; start: Point; origin: Point };

export type PanPhase = Drag["phase"];

const DRAG_THRESHOLD_PX = 4;
const KEY_STEP_PX = 40;

const KEY_DELTA: Partial<Record<string, Point>> = {
  ArrowLeft: { x: KEY_STEP_PX, y: 0 },
  ArrowRight: { x: -KEY_STEP_PX, y: 0 },
  ArrowUp: { x: 0, y: KEY_STEP_PX },
  ArrowDown: { x: 0, y: -KEY_STEP_PX },
};

const ORIGIN: Point = { x: 0, y: 0 };

function clamp(p: Point, max: Point): Point {
  return {
    x: Math.min(0, Math.max(-max.x, p.x)),
    y: Math.min(0, Math.max(-max.y, p.y)),
  };
}

export function usePan(
  viewportRef: RefObject<HTMLDivElement | null>,
  layerRef: RefObject<HTMLDivElement | null>,
) {
  const [drag, setDrag] = useState<Drag>({ phase: "idle" });
  const [raw, setRaw] = useState<Point>(ORIGIN);
  const [max, setMax] = useState<Point>(ORIGIN);
  const [contentHeight, setContentHeight] = useState(0);
  const suppressClick = useRef(false);

  useLayoutEffect(() => {
    const viewport = viewportRef.current;
    const layer = layerRef.current;
    if (!viewport || !layer) {
      return;
    }
    const measure = () => {
      setMax({
        x: Math.max(0, layer.offsetWidth - viewport.clientWidth),
        y: Math.max(0, layer.offsetHeight - viewport.clientHeight),
      });
      setContentHeight(layer.offsetHeight);
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(viewport);
    observer.observe(layer);
    return () => observer.disconnect();
  }, [viewportRef, layerRef]);

  const offset = clamp(raw, max);
  const pannable = max.x > 0 || max.y > 0;
  const panBy = (delta: Point) =>
    setRaw((prev) => {
      const current = clamp(prev, max);
      return clamp({ x: current.x + delta.x, y: current.y + delta.y }, max);
    });

  const viewportProps = {
    onPointerDown: (e: PointerEvent<HTMLDivElement>) => {
      suppressClick.current = false;
      if (e.button !== 0 || !pannable) {
        return;
      }
      setDrag({
        phase: "pressing",
        pointerId: e.pointerId,
        start: { x: e.clientX, y: e.clientY },
        origin: offset,
      });
    },
    onPointerMove: (e: PointerEvent<HTMLDivElement>) => {
      if (drag.phase === "idle" || drag.pointerId !== e.pointerId) {
        return;
      }
      const dx = e.clientX - drag.start.x;
      const dy = e.clientY - drag.start.y;
      if (drag.phase === "pressing") {
        if (Math.hypot(dx, dy) < DRAG_THRESHOLD_PX) {
          return;
        }
        e.currentTarget.setPointerCapture(e.pointerId);
        setDrag({ ...drag, phase: "panning" });
      }
      setRaw(clamp({ x: drag.origin.x + dx, y: drag.origin.y + dy }, max));
    },
    onPointerUp: () => {
      suppressClick.current = drag.phase === "panning";
      setDrag({ phase: "idle" });
    },
    onPointerCancel: () => setDrag({ phase: "idle" }),
    onLostPointerCapture: () => setDrag({ phase: "idle" }),
    onClickCapture: (e: MouseEvent<HTMLDivElement>) => {
      if (suppressClick.current) {
        suppressClick.current = false;
        e.preventDefault();
        e.stopPropagation();
      }
    },
    onDragStart: (e: DragEvent<HTMLDivElement>) => e.preventDefault(),
    onKeyDown: (e: KeyboardEvent<HTMLDivElement>) => {
      const delta = KEY_DELTA[e.key];
      if (e.target !== e.currentTarget || !delta) {
        return;
      }
      e.preventDefault();
      panBy(delta);
    },
    // Focusing a clipped child scrolls the overflow-hidden viewport natively; fold that scroll into the pan offset instead.
    onScroll: (e: UIEvent<HTMLDivElement>) => {
      const el = e.currentTarget;
      if (el.scrollLeft === 0 && el.scrollTop === 0) {
        return;
      }
      panBy({ x: -el.scrollLeft, y: -el.scrollTop });
      el.scrollTo(0, 0);
    },
  };

  return { offset, phase: drag.phase, pannable, contentHeight, viewportProps };
}
