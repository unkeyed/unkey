"use client";

import { cn } from "@unkey/ui";
import {
  AnimatePresence,
  type MotionValue,
  animateSingleValue,
  m,
  useMotionValue,
  useReducedMotion,
  useTransform,
} from "framer-motion";
import { type ReactNode, createContext, use, useEffect, useRef, useState } from "react";
import { CardSurface } from "./card";
import { COLUMN_CARD_MAX_HEIGHT, ColumnGrid, type Role, roleCell } from "./flow-layout";

type Direction = "forward" | "back";

const NEIGHBOR_OPACITY = 0.4;
const ENTER_OFFSET_PX = 64;
const EASE_OUT_EXPO = [0.16, 1, 0.3, 1] as const;
const MOVE_S = 0.62;
const FADE = {
  in: { duration: 0.32, delay: 0.12, ease: "easeOut" },
  out: { duration: 0.2, ease: "easeOut" },
} as const;
const MASK_STOPS = 12;

function easedAlpha(t: number): string {
  return ((1 - Math.cos(Math.PI * t)) / 2).toFixed(3);
}

const MASK_STEPS = Array.from({ length: MASK_STOPS + 1 }, (_, i) => {
  const t = i / MASK_STOPS;
  return { t, topAlpha: easedAlpha(t), bottomAlpha: easedAlpha(1 - t) };
});

function easedMask(top: number, bottom: number): string {
  const topStops = MASK_STEPS.map(
    ({ t, topAlpha }) => `rgb(0 0 0 / ${topAlpha}) ${(top * t).toFixed(1)}px`,
  );
  const bottomStops = MASK_STEPS.map(
    ({ t, bottomAlpha }) =>
      `rgb(0 0 0 / ${bottomAlpha}) calc(100% - ${(bottom * (1 - t)).toFixed(1)}px)`,
  );
  return `linear-gradient(to bottom, ${[...topStops, ...bottomStops].join(", ")})`;
}

function moveTransition(animated: boolean) {
  return animated ? { duration: MOVE_S, ease: EASE_OUT_EXPO } : { duration: 0 };
}

type ColumnMotion = {
  animated: boolean;
  fadeTop: MotionValue<number>;
  fadeBottom: MotionValue<number>;
  heights: Map<string, number>;
};

const ColumnMotionContext = createContext<ColumnMotion | null>(null);

function useColumnMotion(): ColumnMotion {
  const motionState = use(ColumnMotionContext);
  if (!motionState) {
    throw new Error("StepColumn must render inside AnimatedColumn");
  }
  return motionState;
}

type AnimatedColumnProps<Id extends string> = {
  at: number;
  previous: Id | null;
  current: Id;
  next: Id | null;
  renderCard: (id: Id) => ReactNode;
};

export function AnimatedColumn<Id extends string>({ at, ...column }: AnimatedColumnProps<Id>) {
  const animated = useAnimatedAfterInteraction();
  const fadeTop = useMotionValue(0);
  const fadeBottom = useMotionValue(0);
  const [heights] = useState(() => new Map<string, number>());
  const direction = useColumnDirection(at);
  return (
    <ColumnMotionContext.Provider value={{ animated, fadeTop, fadeBottom, heights }}>
      <StepColumn direction={direction} {...column} />
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

function ColumnCard({
  id,
  current,
  children,
}: {
  id: string;
  current: boolean;
  children: ReactNode;
}) {
  const { animated, heights } = useColumnMotion();
  const [live, setLive] = useState(current);
  if (current && !live) {
    setLive(true);
  }
  const ref = useRef<HTMLDivElement>(null);
  return (
    <m.div
      ref={ref}
      className="relative flex min-h-0 flex-col"
      initial={false}
      animate={{ opacity: current ? 1 : 0 }}
      transition={animated ? FADE[current ? "in" : "out"] : { duration: 0 }}
      onAnimationComplete={() => {
        if (!current && ref.current) {
          heights.set(id, ref.current.offsetHeight);
          setLive(false);
        }
      }}
    >
      {live ? children : <div style={{ height: heights.get(id) ?? COLUMN_CARD_MAX_HEIGHT }} />}
    </m.div>
  );
}

type Peek = { top: number; bottom: number };

function usePeek(view: HTMLElement | null, currentId: string): Peek {
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

function useFocusOnMove(view: HTMLElement | null, currentId: string) {
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

function StepColumn<Id extends string>({
  previous,
  current,
  next,
  direction,
  renderCard,
}: Omit<AnimatedColumnProps<Id>, "at"> & { direction: Direction }) {
  const { animated, fadeTop, fadeBottom } = useColumnMotion();
  const transition = moveTransition(animated);
  const [view, setView] = useState<HTMLDivElement | null>(null);
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
    if (performance.now() - moved.current.at > MOVE_S * 1000) {
      fadeTop.jump(topTarget);
      fadeBottom.jump(bottomTarget);
      return;
    }
    const controls = [
      animateSingleValue(fadeTop, topTarget, moveTransition(animated)),
      animateSingleValue(fadeBottom, bottomTarget, moveTransition(animated)),
    ];
    return () => {
      for (const c of controls) {
        // Instant transitions return undefined despite the type.
        c?.stop();
      }
    };
  }, [topTarget, bottomTarget, fadeTop, fadeBottom, animated]);
  const mask = useTransform(() => easedMask(fadeTop.get(), fadeBottom.get()));

  const offset = direction === "forward" ? ENTER_OFFSET_PX : -ENTER_OFFSET_PX;
  const items: { role: Role; id: Id }[] = [
    ...(previous ? [{ role: "previous" as const, id: previous }] : []),
    { role: "current", id: current },
    ...(next ? [{ role: "next" as const, id: next }] : []),
  ];

  return (
    <m.div
      ref={setView}
      className="relative min-h-0 flex-1 overflow-hidden"
      style={{ maskImage: mask, WebkitMaskImage: mask }}
    >
      <ColumnGrid>
        <AnimatePresence initial={false}>
          {items.map(({ role, id }) => (
            <m.div
              key={id}
              layoutId={`card-${id}`}
              layout="position"
              layoutDependency={current}
              data-current-card={role === "current" ? id : undefined}
              inert={role !== "current"}
              className={cn(
                "relative col-start-1 flex min-h-0 flex-col",
                roleCell[role],
                role !== "current" && "pointer-events-none",
              )}
              style={{ maxHeight: COLUMN_CARD_MAX_HEIGHT }}
              initial={{ opacity: 0, y: offset }}
              animate={{ opacity: role === "current" ? 1 : NEIGHBOR_OPACITY, y: 0 }}
              exit={{ opacity: 0, y: -offset }}
              transition={transition}
            >
              <CardSurface className="absolute inset-0" />
              <ColumnCard id={id} current={role === "current"}>
                {renderCard(id)}
              </ColumnCard>
            </m.div>
          ))}
        </AnimatePresence>
      </ColumnGrid>
    </m.div>
  );
}
