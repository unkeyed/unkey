"use client";

import { cn } from "cn";
import {
  type CSSProperties,
  type ReactNode,
  useEffect,
  useLayoutEffect,
  useReducer,
  useRef,
} from "react";
import { type TickerLine, type TickerTone, emptyTicker, tickerReducer } from "./ticker-model";

const TICKER = {
  swapMs: 200,
  easing: "cubic-bezier(0.42, 0, 0.58, 1)",
  dwellMs: 1000,
  maskStops: [7, 35, 64, 93],
  maskPadPx: 6,
};

const tickerTone: Record<TickerTone, string> = {
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
    paddingBlock: TICKER.maskPadPx,
    marginBlock: -TICKER.maskPadPx,
    maskImage: mask,
    WebkitMaskImage: mask,
  };
}

export function LogTicker({ line, fallback }: { line: TickerLine | null; fallback: ReactNode }) {
  const [ticker, dispatch] = useReducer(tickerReducer, { ...emptyTicker, shown: line });
  const outRef = useRef<HTMLSpanElement>(null);
  const inRef = useRef<HTMLSpanElement>(null);

  useEffect(() => {
    dispatch({ type: "receive", line, animate: !prefersReducedMotion() });
  }, [line]);

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
    return fallback;
  }
  return (
    <>
      <span className="block sm:sr-only">{fallback}</span>
      <span
        aria-hidden="true"
        className="hidden max-w-[300px] overflow-hidden sm:block"
        style={maskStyle()}
      >
        <span
          className="relative block h-5 min-w-0 font-mono text-xs leading-5"
          style={{ perspective: 200 }}
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
          {incoming && (
            <span
              key={incoming.id}
              ref={inRef}
              className={cn("absolute inset-0 truncate opacity-0", tickerTone[incoming.tone])}
            >
              {incoming.text}
            </span>
          )}
        </span>
      </span>
    </>
  );
}
