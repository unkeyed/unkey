"use client";

import { isLand } from "@/app/(app)/[workspaceSlug]/projects/_components/region/dotted-map";
import { RectFlag } from "@/app/(app)/[workspaceSlug]/projects/_components/region/rect-flag";
import { regionInfo } from "@/app/(app)/[workspaceSlug]/projects/_components/region/region-info";
import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { useEffect, useMemo, useRef, useState } from "react";

const EASE_OUT = [0.23, 1, 0.32, 1] as const;
const EASE_OUT_EXPO = [0.19, 1, 0.22, 1] as const;
const MAP_IN_S = 0.7;
const PIN_DELAY_S = 0.3;
const PIN_IN_S = 0.6;
const LABEL_LAG_S = 0.1;
const PIN_STAGGER_S = 0.06;
const PING_KEYFRAMES: Keyframe[] = [
  { opacity: 0.16, scale: 1, offset: 0, easing: "cubic-bezier(0.2, 0.6, 0.35, 1)" },
  { opacity: 0, scale: 3.5, offset: 0.65 },
  { opacity: 0, scale: 3.5, offset: 1 },
];
const PING_CYCLE_MS = 2600;

const WORLD = { lonMin: -170, latTop: 64, width: 350, height: 86 };
const DOT_STEP = 2;
const DOT_R = 0.3;

function wrapLon(lon: number) {
  return ((((lon + 180) % 360) + 360) % 360) - 180;
}

function landDots(lonMin: number) {
  const dots: { x: number; y: number }[] = [];
  for (let y = DOT_STEP / 2; y < WORLD.height; y += DOT_STEP) {
    for (let x = DOT_STEP / 2; x < WORLD.width; x += DOT_STEP) {
      if (isLand(wrapLon(x + lonMin), WORLD.latTop - y)) {
        dots.push({ x, y });
      }
    }
  }
  return dots;
}

const WORLD_DOTS = landDots(WORLD.lonMin);

function pinPosition(pin: { lon: number; lat: number }, lonMin: number) {
  return {
    left: `${((pin.lon - lonMin) / WORLD.width) * 100}%`,
    top: `${((WORLD.latTop - pin.lat) / WORLD.height) * 100}%`,
  };
}

function RegionPing({ delayS }: { delayS: number }) {
  const ref = useRef<HTMLSpanElement>(null);
  useEffect(() => {
    const ping = ref.current?.animate(PING_KEYFRAMES, {
      duration: PING_CYCLE_MS,
      delay: delayS * 1000,
      iterations: Number.POSITIVE_INFINITY,
    });
    return () => ping?.cancel();
  }, [delayS]);
  return (
    <span
      ref={ref}
      aria-hidden
      className="absolute size-2.5 -translate-x-1/2 -translate-y-1/2 rounded-full bg-gray-12 opacity-0"
    />
  );
}

const LOCAL_REGION = "local";
const LIVERPOOL = { lon: -2.98, lat: 53.41 };

function regionOnMap(name: string) {
  const info = regionInfo(name);
  return name === LOCAL_REGION
    ? { ...info, pin: LIVERPOOL, labelled: false }
    : { ...info, labelled: true };
}

export function LiveRegionsMap({ regions }: { regions: string[] }) {
  const reduceMotion = useReducedMotion();
  const [hovered, setHovered] = useState<string | null>(null);
  const [pinsIn, setPinsIn] = useState(false);
  const infos = regions.map(regionOnMap);
  const pinned = infos.flatMap((info) => (info.pin ? [{ ...info, pin: info.pin }] : []));
  const offMap = infos.filter((info) => !info.pin);
  const lonMin = pinned.length === 1 ? pinned[0].pin.lon - WORLD.width / 2 : WORLD.lonMin;
  const dots = useMemo(() => (lonMin === WORLD.lonMin ? WORLD_DOTS : landDots(lonMin)), [lonMin]);
  const autoLabel = pinned.length === 1 && pinned[0].labelled ? pinned[0].name : null;
  const hoveredPin = pinned.some((region) => region.name === hovered) ? hovered : null;
  const shown = hoveredPin ?? autoLabel;
  const interactive = pinsIn || reduceMotion === true;
  const enter = (delay: number) =>
    reduceMotion ? { duration: 0 } : { duration: 0.4, ease: EASE_OUT, delay };
  const settle = (delay: number) =>
    reduceMotion ? { duration: 0 } : { duration: PIN_IN_S, ease: EASE_OUT_EXPO, delay };

  return (
    <motion.div
      className="relative w-full"
      style={{ aspectRatio: `${WORLD.width} / ${WORLD.height}` }}
      initial={reduceMotion ? false : { opacity: 0, scale: 0.88, filter: "blur(8px)" }}
      animate={{ opacity: 1, scale: 1, filter: "blur(0px)" }}
      transition={reduceMotion ? { duration: 0 } : { duration: MAP_IN_S, ease: EASE_OUT_EXPO }}
    >
      <svg
        viewBox={`0 0 ${WORLD.width} ${WORLD.height}`}
        className="absolute inset-0 size-full [mask-image:radial-gradient(ellipse_56%_56%_at_50%_50%,black_45%,transparent_100%)]"
        aria-hidden
      >
        {dots.map((d) => (
          <circle key={`${d.x}-${d.y}`} cx={d.x} cy={d.y} r={DOT_R} className="fill-gray-7" />
        ))}
      </svg>
      {pinned.map((region, index) => {
        const delay = PIN_DELAY_S + index * PIN_STAGGER_S;
        return (
          <div key={region.name} className="absolute" style={pinPosition(region.pin, lonMin)}>
            {reduceMotion ? null : <RegionPing delayS={delay + PIN_IN_S} />}
            <motion.div
              aria-hidden
              initial={reduceMotion ? false : { opacity: 0, scale: 0.85 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={settle(delay)}
              onAnimationComplete={index === pinned.length - 1 ? () => setPinsIn(true) : undefined}
            >
              <span className="absolute size-[18px] -translate-x-1/2 -translate-y-1/2 rounded-full bg-grayA-3" />
              <span className="absolute size-2.5 -translate-x-1/2 -translate-y-1/2 rounded-full border border-gray-7 bg-white" />
            </motion.div>
            {region.labelled && interactive ? (
              <span
                className="absolute size-6 -translate-x-1/2 -translate-y-1/2"
                onMouseEnter={() => setHovered(region.name)}
                onMouseLeave={() => setHovered(null)}
              />
            ) : null}
            <AnimatePresence>
              {region.labelled && shown === region.name ? (
                <motion.span
                  className="pointer-events-none absolute bottom-2.5 left-0 flex -translate-x-1/2 items-center gap-1 whitespace-nowrap rounded bg-gray-12 px-1 py-px font-medium text-3xs text-gray-1 shadow-sm"
                  initial={reduceMotion ? false : { opacity: 0, y: 2 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: 2 }}
                  transition={
                    region.name === autoLabel
                      ? settle(delay + LABEL_LAG_S)
                      : { duration: reduceMotion ? 0 : 0.15, ease: EASE_OUT }
                  }
                >
                  <RectFlag flag={region.flag} size="xs" />
                  {region.city}
                </motion.span>
              ) : null}
            </AnimatePresence>
          </div>
        );
      })}
      {offMap.length > 0 ? (
        <div className="absolute bottom-2 left-2 flex gap-1.5">
          {offMap.map((region, index) => (
            <motion.span
              key={region.name}
              className="flex items-center gap-1.5 rounded-md border border-grayA-4 bg-raised px-1.5 py-0.5 font-mono text-2xs text-gray-11"
              initial={reduceMotion ? false : { opacity: 0, y: 4 }}
              animate={{ opacity: 1, y: 0 }}
              transition={enter(PIN_DELAY_S + index * PIN_STAGGER_S)}
            >
              <span className="size-1.5 rounded-full bg-gray-12" />
              {region.name}
            </motion.span>
          ))}
        </div>
      ) : null}
    </motion.div>
  );
}
