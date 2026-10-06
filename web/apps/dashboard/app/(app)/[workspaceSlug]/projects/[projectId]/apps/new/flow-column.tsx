"use client";

import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import {
  AnimatePresence,
  type MotionValue,
  animate,
  motion,
  useMotionTemplate,
  useMotionValue,
  useReducedMotion,
} from "framer-motion";
import { createContext, useContext, useEffect, useRef, useState } from "react";
import { COLUMN_CARD_MAX_HEIGHT, COLUMN_GAP, COLUMN_TOP_SPACE, cardSurface } from "./card";
import { ConfigCard } from "./config-card";
import { useNewAppFlow } from "./flow";
import { Result, Watch } from "./panes/deploying-pane";
import { DeployRunProvider } from "./panes/deploying/use-deploy-run";
import { type CardId, type Direction, previousCard, resolveCard } from "./wizard-model";

type Role = "previous" | "current" | "next";

const NEIGHBOR_OPACITY = 0.4;
const ENTER_OFFSET_PX = 64;
const EASE_IN_OUT_CUBIC = [0.645, 0.045, 0.355, 1] as const;
const MOVE_S = 0.4;
const FADE_S = 0.2;

const roleCell: Record<Role, string> = {
  previous: "row-start-1 self-end",
  current: "row-start-2",
  next: "row-start-3 self-start",
};

type ColumnMotion = {
  animated: boolean;
  fadeTop: MotionValue<number>;
  fadeBottom: MotionValue<number>;
  heights: Map<CardId, number>;
};

const ColumnMotionContext = createContext<ColumnMotion | null>(null);

function useColumnMotion(): ColumnMotion {
  const motionState = useContext(ColumnMotionContext);
  if (!motionState) {
    throw new Error("StepColumn must render inside FlowColumn");
  }
  return motionState;
}

// A deployment wraps the column in DeployRunProvider, which remounts it; the
// motion state lives here so that swap animates like any other step.
export function FlowColumn() {
  const { state, projectId, dispatch } = useNewAppFlow();
  const animated = useAnimatedAfterInteraction();
  const fadeTop = useMotionValue(0);
  const fadeBottom = useMotionValue(0);
  const [heights] = useState(() => new Map<CardId, number>());
  const { app, deploymentId } = state;
  return (
    <ColumnMotionContext.Provider value={{ animated, fadeTop, fadeBottom, heights }}>
      {app && deploymentId ? (
        <DeployRunProvider
          projectId={projectId}
          appId={app.id}
          source={app.source}
          deploymentId={deploymentId}
          onDeploymentCreated={(id) => dispatch({ type: "deployment-created", deploymentId: id })}
        >
          <Column />
        </DeployRunProvider>
      ) : (
        <Column />
      )}
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
  const { card, cards } = useNewAppFlow();
  const at = cards.indexOf(card.id);
  const direction = useColumnDirection(at);
  return (
    <StepColumn
      direction={direction}
      previous={cards[at - 1] ?? null}
      current={card.id}
      next={cards[at + 1] ?? null}
    />
  );
}

function CardContent({ id }: { id: CardId }) {
  const { state, locked, dispatch } = useNewAppFlow();
  const at = { ...state, card: id };
  const card = resolveCard(at);
  if (card.id !== id) {
    return null;
  }
  if (card.id === "watch") {
    return <Watch appId={card.appId} />;
  }
  if (card.id === "result") {
    return <Result />;
  }
  const previous = previousCard(at);
  return (
    <ConfigCard
      card={card}
      back={
        previous && !locked ? (
          <Button
            variant="outline"
            size="sm"
            onClick={() => dispatch({ type: "go", card: previous })}
          >
            Back
          </Button>
        ) : null
      }
    />
  );
}

// Only the current card mounts its pane, so neighbours run no queries. A leaving
// card keeps its content until the fade ends, then holds its height as a shell.
function ColumnCard({ id, current }: { id: CardId; current: boolean }) {
  const { animated, heights } = useColumnMotion();
  const [live, setLive] = useState(current);
  if (current && !live) {
    setLive(true);
  }
  const ref = useRef<HTMLDivElement>(null);
  return (
    <motion.div
      ref={ref}
      className="relative flex min-h-0 flex-col"
      initial={false}
      animate={{ opacity: current ? 1 : 0 }}
      transition={animated ? { duration: FADE_S, ease: "easeOut" } : { duration: 0 }}
      onAnimationComplete={() => {
        if (!current && ref.current) {
          heights.set(id, ref.current.offsetHeight);
          setLive(false);
        }
      }}
    >
      {live ? (
        <CardContent id={id} />
      ) : (
        <div style={{ height: heights.get(id) ?? COLUMN_CARD_MAX_HEIGHT }} />
      )}
    </motion.div>
  );
}

type Peek = { top: number; bottom: number };

function usePeek(view: HTMLElement | null, currentId: CardId): Peek {
  const [peek, setPeek] = useState<Peek>({ top: 0, bottom: 0 });
  // biome-ignore lint/correctness/useExhaustiveDependencies: re-measure when a new card takes the middle.
  useEffect(() => {
    const current = view?.querySelector<HTMLElement>("[data-current-card]");
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

// A resumed flow settles its stage as data loads, which can take seconds (the
// repository list after a GitHub install). Only moves the user starts should
// glide, so the column stays instant until the first pointer or key press.
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
  previous: CardId | null;
  current: CardId;
  next: CardId | null;
  direction: Direction;
}) {
  const { animated, fadeTop, fadeBottom } = useColumnMotion();
  const transition = animated ? { duration: MOVE_S, ease: EASE_IN_OUT_CUBIC } : { duration: 0 };
  const [view, setView] = useState<HTMLDivElement | null>(null);
  const peek = usePeek(view, current);

  const moved = useRef({ id: current, at: 0 });
  if (moved.current.id !== current) {
    moved.current = { id: current, at: performance.now() };
  }

  const topTarget = previous ? peek.top : 0;
  const bottomTarget = next ? peek.bottom : 0;
  // biome-ignore lint/correctness/useExhaustiveDependencies: transition only changes with reduced motion.
  useEffect(() => {
    // Content loading inside a card resizes it; only a step change should glide.
    if (performance.now() - moved.current.at > MOVE_S * 1000) {
      fadeTop.jump(topTarget);
      fadeBottom.jump(bottomTarget);
      return;
    }
    const controls = [
      animate(fadeTop, topTarget, transition),
      animate(fadeBottom, bottomTarget, transition),
    ];
    return () => {
      for (const c of controls) {
        c.stop();
      }
    };
  }, [topTarget, bottomTarget, fadeTop, fadeBottom]);
  const mask = useMotionTemplate`linear-gradient(to bottom, transparent 0, black ${fadeTop}px, black calc(100% - ${fadeBottom}px), transparent 100%)`;

  const offset = direction === "forward" ? ENTER_OFFSET_PX : -ENTER_OFFSET_PX;
  const items: { role: Role; id: CardId }[] = [
    ...(previous ? [{ role: "previous" as const, id: previous }] : []),
    { role: "current", id: current },
    ...(next ? [{ role: "next" as const, id: next }] : []),
  ];

  return (
    <motion.div
      ref={setView}
      className="relative min-h-0 flex-1 overflow-hidden"
      style={{ maskImage: mask, WebkitMaskImage: mask }}
    >
      <div
        className="grid h-full grid-cols-[minmax(0,560px)] justify-center px-8"
        style={{
          rowGap: COLUMN_GAP,
          gridTemplateRows: `${COLUMN_TOP_SPACE} auto minmax(0, 1fr)`,
        }}
      >
        <AnimatePresence initial={false}>
          {items.map(({ role, id }) => (
            <motion.div
              key={id}
              layoutId={`new-app-card-${id}`}
              layout="position"
              layoutDependency={current}
              data-current-card={role === "current" ? "" : undefined}
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
              <div aria-hidden className={cn(cardSurface, "absolute inset-0")} />
              <ColumnCard id={id} current={role === "current"} />
            </motion.div>
          ))}
        </AnimatePresence>
      </div>
    </motion.div>
  );
}
