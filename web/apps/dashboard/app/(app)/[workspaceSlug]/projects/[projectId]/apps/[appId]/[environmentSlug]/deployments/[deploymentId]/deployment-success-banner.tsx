"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { useEffect, useState } from "react";
import { createPortal } from "react-dom";

const VICTORY_DURATION_SECONDS = 4.5;
const ZOOM_BLUR_SIZE = 1.12;
const ZOOM_BLUR_STEPS = 22;
const ZOOM_BLUR_LAYERS = Array.from({ length: ZOOM_BLUR_STEPS + 1 }, (_, index) => {
  const step = ZOOM_BLUR_STEPS - index;
  const progress = step / ZOOM_BLUR_STEPS;
  return {
    opacity: 0.05 * 2 ** -progress,
    scale: ZOOM_BLUR_SIZE ** progress,
  };
});

const textClassName =
  "absolute left-[49.6%] top-[72.5%] whitespace-nowrap text-center text-[9.6296dvh] font-normal uppercase leading-none tracking-[0.463dvh] [font-family:'Adobe_Garamond_Pro','adobe-garamond-pro',Garamond,'Times_New_Roman',serif]";

export function DeploymentSuccessBanner({ visible }: { visible: boolean }) {
  const reduceMotion = useReducedMotion();
  const [mounted, setMounted] = useState(false);

  useEffect(() => setMounted(true), []);

  if (!mounted) {
    return null;
  }

  return createPortal(
    <AnimatePresence>
      {visible && (
        <motion.output
          className="pointer-events-none fixed inset-0 z-[100] block overflow-hidden"
          initial={{ opacity: 0 }}
          animate={reduceMotion ? { opacity: 1 } : { opacity: [0, 1, 1, 0] }}
          exit={{ opacity: 0 }}
          transition={
            reduceMotion
              ? { duration: 0 }
              : {
                  duration: VICTORY_DURATION_SECONDS,
                  times: [0, 0.18, 0.78, 1],
                  ease: [0.4, 0, 0.2, 1],
                }
          }
          aria-live="polite"
        >
          <div className="absolute inset-0 bg-black/35" />
          <div className="absolute left-[-17.5dvh] top-[54.6%] h-[35%] w-[calc(100%+35dvh)] bg-[linear-gradient(to_bottom,transparent_0%,rgba(0,0,0,0.9)_25%,rgba(0,0,0,0.9)_75%,transparent_100%)] blur-[1.75dvh]" />
          {ZOOM_BLUR_LAYERS.map((layer, index) => (
            <span
              // biome-ignore lint/suspicious/noArrayIndexKey: layers are a static ordered visual effect.
              key={index}
              aria-hidden="true"
              className={textClassName}
              style={{
                color: `rgba(254, 109, 40, ${layer.opacity})`,
                filter: "blur(max(1px, 0.0926dvh))",
                transform: `translate(-50%, -50%) scale(${layer.scale}) scaleY(1.5)`,
              }}
            >
              Application Deployed
            </span>
          ))}
          <span
            className={textClassName}
            style={{
              color: "rgba(255, 210, 87, 0.9)",
              transform: "translate(-50%, -50%) scaleY(1.5)",
            }}
          >
            Application Deployed
          </span>
        </motion.output>
      )}
    </AnimatePresence>,
    document.body,
  );
}
