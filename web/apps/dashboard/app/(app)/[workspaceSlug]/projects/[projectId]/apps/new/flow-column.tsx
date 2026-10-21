"use client";

import { useLayoutMotion } from "@/hooks/use-layout-motion";
import { Button, cn } from "@unkey/ui";
import { createContext, use, useCallback, useEffect, useRef, useState } from "react";
import { CardSurface } from "./card";
import { ConfigCard } from "./config-card";
import { useNewAppFlow } from "./flow";
import { COLUMN_CARD_MAX_HEIGHT, ColumnGrid, type Role, roleCell } from "./flow-layout";
import { type Direction, type Pane, cardView, columnCards } from "./wizard-model";

const NEIGHBOR_OPACITY = 0.4;
const ENTER_OFFSET_PX = 64;
const EASE_OUT_EXPO = "cubic-bezier(0.16, 1, 0.3, 1)";
const MOVE_MS = 620;
const FADE = {
  in: "opacity 320ms ease-out 120ms",
  out: "opacity 200ms ease-out",
} as const;
const MASK_STOPS = 12;

function easedAlpha(t: number): string {
  return ((1 - Math.cos(Math.PI * t)) / 2).toFixed(3);
}

// The fade lengths are registered custom properties (see styles/tailwind.css), so a CSS
// transition on them animates the whole mask.
const COLUMN_MASK = (() => {
  const steps = Array.from({ length: MASK_STOPS + 1 }, (_, i) => i / MASK_STOPS);
  const topStops = steps.map(
    (t) => `rgb(0 0 0 / ${easedAlpha(t)}) calc(var(--column-fade-top) * ${t.toFixed(4)})`,
  );
  const bottomStops = steps.map(
    (t) =>
      `rgb(0 0 0 / ${easedAlpha(1 - t)}) calc(100% - var(--column-fade-bottom) * ${(1 - t).toFixed(4)})`,
  );
  return `linear-gradient(to bottom, ${[...topStops, ...bottomStops].join(", ")})`;
})();

type ColumnMotion = {
  animated: boolean;
  heights: Map<Pane, number>;
};

const ColumnMotionContext = createContext<ColumnMotion | null>(null);

function useColumnMotion(): ColumnMotion {
  const motionState = use(ColumnMotionContext);
  if (!motionState) {
    throw new Error("StepColumn must render inside FlowColumn");
  }
  return motionState;
}

export function FlowColumn() {
  const animated = useAnimatedAfterInteraction();
  const [heights] = useState(() => new Map<Pane, number>());
  return (
    <ColumnMotionContext.Provider value={{ animated, heights }}>
      <Column />
    </ColumnMotionContext.Provider>
  );
}

function useColumnDirection(index: number): Direction {
  const [last, setLast] = useState<{ index: number; direction: Direction }>({
    index,
    direction: "forward",
  });
  if (last.index !== index) {
    const direction = index > last.index ? "forward" : "back";
    setLast({ index, direction });
    return direction;
  }
  return last.direction;
}

function Column() {
  const { state } = useNewAppFlow();
  const { at, previous, current, next } = columnCards(state);
  const direction = useColumnDirection(at);
  return <StepColumn direction={direction} previous={previous} current={current} next={next} />;
}

function CardContent({ id }: { id: Pane }) {
  const { state, locked, dispatch } = useNewAppFlow();
  const view = cardView(state, id);
  if (!view) {
    return null;
  }
  const { previous } = view;
  return (
    <ConfigCard
      view={view}
      back={
        previous ? (
          <Button
            variant="outline"
            size="sm"
            disabled={locked}
            onClick={() => dispatch({ type: "go", step: previous })}
          >
            Back
          </Button>
        ) : null
      }
    />
  );
}

function ColumnCard({ id, current }: { id: Pane; current: boolean }) {
  const { animated, heights } = useColumnMotion();
  const [live, setLive] = useState(current);
  if (current && !live) {
    setLive(true);
  }
  const ref = useRef<HTMLDivElement>(null);
  const retire = useCallback(() => {
    if (ref.current) {
      heights.set(id, ref.current.offsetHeight);
    }
    setLive(false);
  }, [heights, id]);
  useEffect(() => {
    if (!current && live && !animated) {
      retire();
    }
  }, [current, live, animated, retire]);
  return (
    <div
      ref={ref}
      className="relative flex min-h-0 flex-col"
      style={{
        opacity: current ? 1 : 0,
        transition: animated ? FADE[current ? "in" : "out"] : "none",
      }}
      onTransitionEnd={(event) => {
        if (!current && event.target === ref.current && event.propertyName === "opacity") {
          retire();
        }
      }}
    >
      {live ? (
        <CardContent id={id} />
      ) : (
        <div style={{ height: heights.get(id) ?? COLUMN_CARD_MAX_HEIGHT }} />
      )}
    </div>
  );
}

type Peek = { top: number; bottom: number };

function usePeek(view: HTMLElement | null, currentId: Pane): Peek {
  const [peek, setPeek] = useState<Peek>({ top: 0, bottom: 0 });
  useEffect(() => {
    const current = view?.querySelector<HTMLElement>(`[data-current-card="${currentId}"]`);
    if (!view || !current) {
      return;
    }
    const measure = () =>
      setPeek({
        top: current.offsetTop,
        bottom: Math.max(0, view.clientHeight - current.offsetTop - current.offsetHeight),
      });
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(view);
    observer.observe(current);
    return () => observer.disconnect();
  }, [view, currentId]);
  return peek;
}

function useFocusOnMove(view: HTMLElement | null, currentId: Pane) {
  const focused = useRef(currentId);
  useEffect(() => {
    if (focused.current === currentId) {
      return;
    }
    focused.current = currentId;
    const frame = requestAnimationFrame(() => {
      const card = view?.querySelector<HTMLElement>(`[data-current-card="${currentId}"]`);
      if (card && !card.contains(document.activeElement)) {
        card.querySelector<HTMLElement>("h1")?.focus({ preventScroll: true });
      }
    });
    return () => cancelAnimationFrame(frame);
  }, [view, currentId]);
}

function useReducedMotion() {
  const [reduced, setReduced] = useState(false);
  useEffect(() => {
    const query = window.matchMedia("(prefers-reduced-motion: reduce)");
    const update = () => setReduced(query.matches);
    update();
    query.addEventListener("change", update);
    return () => query.removeEventListener("change", update);
  }, []);
  return reduced;
}

function useAnimatedAfterInteraction() {
  const reduceMotion = useReducedMotion();
  const [interacted, setInteracted] = useState(false);
  useEffect(() => {
    if (interacted) {
      return;
    }
    const controller = new AbortController();
    const options = { capture: true, once: true, signal: controller.signal };
    const settle = () => setInteracted(true);
    window.addEventListener("pointerdown", settle, options);
    window.addEventListener("keydown", settle, options);
    return () => controller.abort();
  }, [interacted]);
  return interacted && !reduceMotion;
}

function StepColumn({
  previous,
  current,
  next,
  direction,
}: {
  previous: Pane | null;
  current: Pane;
  next: Pane | null;
  direction: Direction;
}) {
  const { animated } = useColumnMotion();
  const viewRef = useRef<HTMLDivElement | null>(null);
  const [view, setView] = useState<HTMLDivElement | null>(null);
  const attachView = useCallback((node: HTMLDivElement | null) => {
    viewRef.current = node;
    setView(node);
  }, []);
  const peek = usePeek(view, current);

  const moved = useRef({ id: current, at: 0 });
  useEffect(() => {
    if (moved.current.id !== current) {
      moved.current = { id: current, at: performance.now() };
    }
  }, [current]);

  useFocusOnMove(view, current);

  const topTarget = previous ? peek.top : 0;
  const bottomTarget = next ? peek.bottom : 0;
  useEffect(() => {
    if (!view) {
      return;
    }
    const glide = animated && performance.now() - moved.current.at <= MOVE_MS;
    view.style.transition = glide
      ? `--column-fade-top ${MOVE_MS}ms ${EASE_OUT_EXPO}, --column-fade-bottom ${MOVE_MS}ms ${EASE_OUT_EXPO}`
      : "none";
    view.style.setProperty("--column-fade-top", `${topTarget}px`);
    view.style.setProperty("--column-fade-bottom", `${bottomTarget}px`);
  }, [view, topTarget, bottomTarget, animated]);

  const offset = direction === "forward" ? ENTER_OFFSET_PX : -ENTER_OFFSET_PX;
  const moveTiming = { duration: MOVE_MS, easing: EASE_OUT_EXPO };
  useLayoutMotion(viewRef, {
    enabled: animated,
    appear: false,
    dependency: current,
    move: moveTiming,
    enter: () => ({
      keyframes: [{ opacity: 0, translate: `0 ${offset}px`, offset: 0 }],
      timing: moveTiming,
    }),
    exit: () => ({
      keyframes: [{ opacity: 0, translate: `0 ${-offset}px` }],
      timing: moveTiming,
    }),
  });

  const items: { role: Role; id: Pane }[] = [
    ...(previous ? [{ role: "previous" as const, id: previous }] : []),
    { role: "current", id: current },
    ...(next ? [{ role: "next" as const, id: next }] : []),
  ];

  return (
    <div
      ref={attachView}
      className="relative min-h-0 flex-1 overflow-hidden"
      style={{ maskImage: COLUMN_MASK, WebkitMaskImage: COLUMN_MASK }}
    >
      <ColumnGrid>
        {items.map(({ role, id }) => (
          <div
            key={id}
            data-motion-key={id}
            data-current-card={role === "current" ? id : undefined}
            inert={role !== "current"}
            className={cn(
              "relative col-start-1 flex min-h-0 flex-col",
              roleCell[role],
              role !== "current" && "pointer-events-none",
            )}
            style={{
              maxHeight: COLUMN_CARD_MAX_HEIGHT,
              opacity: role === "current" ? 1 : NEIGHBOR_OPACITY,
              transition: animated ? `opacity ${MOVE_MS}ms ${EASE_OUT_EXPO}` : "none",
            }}
          >
            <CardSurface className="absolute inset-0" />
            <ColumnCard id={id} current={role === "current"} />
          </div>
        ))}
      </ColumnGrid>
    </div>
  );
}
