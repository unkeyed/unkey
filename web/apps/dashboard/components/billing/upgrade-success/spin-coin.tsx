"use client";

import { type ReactNode, useEffect, useRef } from "react";

const INTRO_END_DEG = 900;
const TURNS_PER_CLICK_DEG = 720;
const SPIN_TRANSITION = "transform 1100ms cubic-bezier(0.3, 0.6, 0.25, 1)";

export function SpinCoin({
  front,
  back,
}: {
  front: ReactNode;
  back: ReactNode;
}) {
  const innerRef = useRef<HTMLSpanElement>(null);
  const state = useRef({ ready: false, angle: 0 });

  useEffect(() => {
    const inner = innerRef.current;
    if (!inner) {
      return;
    }
    const onIntroEnd = (event: AnimationEvent) => {
      if (event.target !== inner || !event.animationName.includes("coin-flip")) {
        return;
      }
      inner.style.animation = "none";
      state.current = { ready: true, angle: INTRO_END_DEG };
      inner.style.transform = `rotateY(${INTRO_END_DEG}deg)`;
    };
    inner.addEventListener("animationend", onIntroEnd);
    return () => inner.removeEventListener("animationend", onIntroEnd);
  }, []);

  const spin = () => {
    const inner = innerRef.current;
    const s = state.current;
    if (!inner || !s.ready) {
      return;
    }
    s.angle += TURNS_PER_CLICK_DEG + (Math.random() < 0.5 ? 0 : 180);
    inner.style.transition = SPIN_TRANSITION;
    inner.style.transform = `rotateY(${s.angle}deg)`;
  };

  return (
    <button
      type="button"
      tabIndex={-1}
      onClick={spin}
      aria-label="Spin"
      className="block cursor-pointer rounded-full outline-none perspective-[600px]"
    >
      <span
        ref={innerRef}
        className="relative block size-16 animate-coin-flip transform-3d motion-reduce:animate-none motion-reduce:rotate-y-180"
      >
        <span className="absolute inset-0 backface-hidden flex items-center justify-center rounded-full border border-grayA-4 bg-background shadow-[0_1px_2px_rgba(0,0,0,0.08),0_8px_24px_rgba(0,0,0,0.12)]">
          {front}
        </span>
        <span className="absolute inset-0 rotate-y-180 backface-hidden flex items-center justify-center rounded-full border border-grayA-4 bg-background shadow-[0_1px_2px_rgba(0,0,0,0.08),0_8px_24px_rgba(0,0,0,0.12)]">
          {back}
        </span>
      </span>
    </button>
  );
}
