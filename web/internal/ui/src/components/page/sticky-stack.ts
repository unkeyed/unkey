import type { CSSProperties } from "react";

const STICKY_ORDER = ["header", "controls", "filters", "chart", "columns"] as const;

export type StickySlotName = (typeof STICKY_ORDER)[number];

export const STICKY_ROOT_SELECTOR = '[data-width="full"]';

type StickyGate = "chrome" | "columns";

const STICKY_GATE: Record<
  StickyGate,
  { slotClassName: string; rootEnableClassName: string; enabledFactor: string }
> = {
  chrome: {
    slotClassName:
      "group-data-[width=full]/page:pinned:sticky group-data-[width=full]/page:pinned:top-(--sticky-top) group-data-[width=full]/page:pinned:z-(--sticky-z) group-data-[width=full]/page:pinned:bg-background",
    rootEnableClassName: "data-[width=full]:pinned:[--sticky-chrome-on:1]",
    enabledFactor: "var(--sticky-chrome-on,0)",
  },
  columns: {
    slotClassName:
      "group-data-[width=full]/page:md:sticky group-data-[width=full]/page:md:top-(--sticky-top) group-data-[width=full]/page:md:z-(--sticky-z)",
    rootEnableClassName: "data-[width=full]:md:[--sticky-columns-on:1]",
    enabledFactor: "var(--sticky-columns-on,0)",
  },
};

const SLOT_GATE: Record<StickySlotName, StickyGate> = {
  header: "chrome",
  controls: "chrome",
  filters: "chrome",
  chart: "chrome",
  columns: "columns",
};

export const STICKY_ROOT_CLASS_NAME = Object.values(STICKY_GATE)
  .map((gate) => gate.rootEnableClassName)
  .join(" ");

export function stickySlotHeightVar(slot: StickySlotName) {
  return `--sticky-${slot}-h` as const;
}

function slotHeightTerm(slot: StickySlotName): string {
  return `${STICKY_GATE[SLOT_GATE[slot]].enabledFactor} * var(${stickySlotHeightVar(slot)},0px)`;
}

function stickyTop(slot: StickySlotName): string {
  const slotsAbove = STICKY_ORDER.slice(0, STICKY_ORDER.indexOf(slot));
  return `calc(${["var(--app-header-h,0px)", ...slotsAbove.map(slotHeightTerm)].join(" + ")})`;
}

function stickyZ(slot: StickySlotName): number {
  return 10 + STICKY_ORDER.length - STICKY_ORDER.indexOf(slot);
}

export const STICKY_CHROME_BOTTOM = stickyTop("columns");

export const STICKY_STACK_H = `calc(${STICKY_ORDER.map(slotHeightTerm).join(" + ")})`;

export function stickySlotProps(slot: StickySlotName): { className: string; style: CSSProperties } {
  return {
    className: STICKY_GATE[SLOT_GATE[slot]].slotClassName,
    style: { "--sticky-top": stickyTop(slot), "--sticky-z": stickyZ(slot) } as CSSProperties,
  };
}

export function warnIfSlotOutOfPlace(slot: StickySlotName, element: HTMLElement) {
  if (window.scrollY !== 0) {
    return;
  }
  const style = getComputedStyle(element);
  if (style.position !== "sticky") {
    return;
  }
  const offset = element.getBoundingClientRect().top - Number.parseFloat(style.top);
  if (offset < -1) {
    console.warn(
      `Sticky slot "${slot}" sits ${offset.toFixed(1)}px from its pinned top at scroll 0. Slots must render in the order ${STICKY_ORDER.join(", ")}, with no unregistered content between them.`,
    );
  }
}
