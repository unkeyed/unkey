"use client";

import { cn } from "@unkey/ui/src/lib/utils";
import { type CSSProperties, useEffect, useLayoutEffect, useReducer, useRef } from "react";
import type { LogTone, Stage } from "./run-model";
import { type TickerLine, emptyTicker, stageTickerLine, tickerReducer } from "./ticker-model";
import type { DeployRun } from "./use-deploy-run";

const TICKER = {
  swapMs: 200,
  easing: "cubic-bezier(0.42, 0, 0.58, 1)",
  dwellMs: 1000,
  maskStops: [7, 35, 64, 93],
  maskPadPx: 6,
  lineHeightPx: 20,
  fontSizeClass: "text-xs",
  maxWidthPx: 300,
};

export function LogTicker({ run, stage }: { run: DeployRun; stage: Stage }) {
  return (
    <TickerView line={stageTickerLine(stage, run.groups, run.view.firstError, run.instances)} />
  );
}

const tickerTone: Record<LogTone, string> = {
  plain: "text-gray-11",
  error: "text-error-11",
  warn: "text-warning-11",
};

function prefersReducedMotion(): boolean {
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

function maskStyle(): CSSProperties {
  const [topIn, topFull, bottomFull, bottomOut] = TICKER.maskStops;
  const mask = `linear-gradient(transparent ${topIn}%, #000 ${topFull}%, #000 ${bottomFull}%, transparent ${bottomOut}%)`;
  return {
    maxWidth: TICKER.maxWidthPx,
    paddingBlock: TICKER.maskPadPx,
    marginBlock: -TICKER.maskPadPx,
    maskImage: mask,
    WebkitMaskImage: mask,
  };
}

function TickerView({ line }: { line: TickerLine | null }) {
  const [ticker, dispatch] = useReducer(tickerReducer, { ...emptyTicker, shown: line });
  const outRef = useRef<HTMLSpanElement>(null);
  const inRef = useRef<HTMLSpanElement>(null);

  // biome-ignore lint/correctness/useExhaustiveDependencies: the line id is the trigger.
  useEffect(() => {
    dispatch({ type: "receive", line, animate: !prefersReducedMotion() });
  }, [line?.id]);

  useLayoutEffect(() => {
    const outgoing = outRef.current;
    if (!ticker.leaving || !outgoing) {
      return;
    }
    const fadeOut = outgoing.animate([{ opacity: 1 }, { opacity: 0 }], {
      duration: TICKER.swapMs,
      easing: TICKER.easing,
      fill: "forwards",
    });
    fadeOut.onfinish = () => dispatch({ type: "settle" });
    return () => fadeOut.cancel();
  }, [ticker.leaving]);

  useLayoutEffect(() => {
    const outgoing = outRef.current;
    const incoming = inRef.current;
    if (!ticker.incoming || !outgoing || !incoming) {
      return;
    }
    const { swapMs, easing, dwellMs } = TICKER;
    const timing = { duration: swapMs, easing, fill: "forwards" as const };
    const exit = outgoing.animate(
      [
        { transform: "rotateX(0deg)", transformOrigin: "50% 0%", opacity: 1 },
        { transform: "rotateX(80deg)", transformOrigin: "50% 0%", opacity: 0 },
      ],
      timing,
    );
    const enter = incoming.animate(
      [
        { transform: "rotateX(-80deg)", transformOrigin: "50% 100%", opacity: 0 },
        { transform: "rotateX(0deg)", transformOrigin: "50% 100%", opacity: 1 },
      ],
      timing,
    );
    let dwell: ReturnType<typeof setTimeout> | undefined;
    enter.onfinish = () => {
      dwell = setTimeout(() => dispatch({ type: "settle" }), dwellMs);
    };
    return () => {
      clearTimeout(dwell);
      exit.cancel();
      enter.cancel();
    };
  }, [ticker.incoming]);

  const { shown, incoming } = ticker;
  if (!shown) {
    return null;
  }
  const lineBox: CSSProperties = {
    perspective: 200,
    height: TICKER.lineHeightPx,
    lineHeight: `${TICKER.lineHeightPx}px`,
  };
  return (
    <span className="hidden overflow-hidden sm:block" style={maskStyle()}>
      <span
        className={cn("relative block min-w-0 font-mono", TICKER.fontSizeClass)}
        style={lineBox}
      >
        <span
          key={shown.id}
          ref={outRef}
          title={shown.text}
          className={cn(
            "absolute inset-0 truncate",
            tickerTone[shown.tone],
            shown.waiting && "animate-pulse motion-reduce:animate-none",
          )}
        >
          {shown.text}
        </span>
        {incoming ? (
          <span
            key={incoming.id}
            ref={inRef}
            aria-hidden="true"
            className={cn("absolute inset-0 truncate opacity-0", tickerTone[incoming.tone])}
          >
            {incoming.text}
          </span>
        ) : null}
      </span>
    </span>
  );
}
