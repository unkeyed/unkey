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
import { type ReactNode, createContext, useContext, useEffect, useRef, useState } from "react";
import { COLUMN_CARD_MAX_HEIGHT, COLUMN_GAP, COLUMN_TOP_SPACE, cardSurface } from "./card";
import { ConfigCard } from "./config-card";
import { useNewAppFlow } from "./flow";
import { Result, Watch } from "./panes/deploying-pane";
import { DeployRunProvider } from "./panes/deploying/use-deploy-run";
import {
  type Card,
  type CardId,
  type Direction,
  isConfigCard,
  previousCard,
  resolveCard,
} from "./wizard-model";

type Role = "previous" | "current" | "next";
type ColumnItem = { id: CardId; node: ReactNode };

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
  const { app, deploymentId } = state;
  return (
    <ColumnMotionContext.Provider value={{ animated, fadeTop, fadeBottom }}>
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
  const item = (id: CardId | undefined): ColumnItem | null =>
    id ? { id, node: <NeighbourCard id={id} /> } : null;
  return (
    <StepColumn
      direction={direction}
      previous={item(cards[at - 1])}
      current={{ id: card.id, node: <CurrentCard card={card} /> }}
      next={item(cards[at + 1])}
    />
  );
}

function CurrentCard({ card }: { card: Card }) {
  const { state, locked, dispatch } = useNewAppFlow();
  if (!isConfigCard(card)) {
    return card.id === "watch" ? <Watch appId={card.appId} /> : <Result />;
  }
  const previous = previousCard(state);
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

const placeholder = <div className="h-48" />;

// Neighbours render their real content invisibly so each card keeps its true
// height while it moves; only the empty card surface shows.
function NeighbourCard({ id }: { id: CardId }) {
  const { state } = useNewAppFlow();
  const card = resolveCard({ ...state, card: id, focus: null });
  if (card.id !== id) {
    return placeholder;
  }
  if (!isConfigCard(card)) {
    return card.id === "watch" ? <Watch appId={card.appId} /> : <Result />;
  }
  return (
    <ConfigCard
      card={card}
      back={
        previousCard({ ...state, card: id }) ? (
          <Button variant="outline" size="sm" tabIndex={-1}>
            Back
          </Button>
        ) : null
      }
    />
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
  previous: ColumnItem | null;
  current: ColumnItem;
  next: ColumnItem | null;
  direction: Direction;
}) {
  const { animated, fadeTop, fadeBottom } = useColumnMotion();
  const transition = animated ? { duration: MOVE_S, ease: EASE_IN_OUT_CUBIC } : { duration: 0 };
  const [view, setView] = useState<HTMLDivElement | null>(null);
  const peek = usePeek(view, current.id);

  const moved = useRef({ id: current.id, at: 0 });
  if (moved.current.id !== current.id) {
    moved.current = { id: current.id, at: performance.now() };
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
  const contentTransition = animated
    ? { duration: FADE_S, ease: "easeOut" as const }
    : { duration: 0 };
  const items: { role: Role; item: ColumnItem }[] = [
    ...(previous ? [{ role: "previous" as const, item: previous }] : []),
    { role: "current", item: current },
    ...(next ? [{ role: "next" as const, item: next }] : []),
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
          {items.map(({ role, item }) => (
            <motion.div
              key={item.id}
              layoutId={`new-app-card-${item.id}`}
              layout="position"
              layoutDependency={current.id}
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
              <motion.div
                className="relative flex min-h-0 flex-col"
                initial={false}
                animate={{ opacity: role === "current" ? 1 : 0 }}
                transition={contentTransition}
              >
                {item.node}
              </motion.div>
            </motion.div>
          ))}
        </AnimatePresence>
      </div>
    </motion.div>
  );
}
