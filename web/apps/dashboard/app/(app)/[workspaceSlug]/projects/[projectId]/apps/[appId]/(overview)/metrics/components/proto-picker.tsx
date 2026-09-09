"use client";

import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import "./proto-picker.css";

type Props = {
  variants: string[];
  active: number;
  onChange: (index: number) => void;
};

export function ProtoPicker({ variants, active, onChange }: Props) {
  const navRef = useRef<HTMLElement>(null);
  const highlightRef = useRef<HTMLSpanElement>(null);
  const itemRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const [ready, setReady] = useState(false);

  const moveHighlight = useCallback(() => {
    const el = itemRefs.current[active];
    const highlight = highlightRef.current;
    if (!el || !highlight) {
      return;
    }
    highlight.style.width = `${el.offsetWidth}px`;
    highlight.style.transform = `translateX(${el.offsetLeft}px)`;
  }, [active]);

  useLayoutEffect(() => {
    moveHighlight();
  }, [moveHighlight]);

  useEffect(() => {
    window.addEventListener("resize", moveHighlight);
    const raf = requestAnimationFrame(() => requestAnimationFrame(() => setReady(true)));
    return () => {
      window.removeEventListener("resize", moveHighlight);
      cancelAnimationFrame(raf);
    };
  }, [moveHighlight]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement | null;
      if (
        target &&
        (/^(INPUT|TEXTAREA|SELECT)$/.test(target.tagName) || target.isContentEditable)
      ) {
        return;
      }
      if (e.metaKey || e.ctrlKey || e.altKey) {
        return;
      }
      const num = Number.parseInt(e.key, 10);
      if (num >= 1 && num <= variants.length) {
        onChange(num - 1);
      } else if (e.key === "ArrowRight") {
        onChange((active + 1) % variants.length);
      } else if (e.key === "ArrowLeft") {
        onChange((active - 1 + variants.length) % variants.length);
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [active, variants.length, onChange]);

  return (
    <nav
      ref={navRef}
      className="proto-picker"
      aria-label="Prototype variants"
      data-ready={ready ? "" : undefined}
    >
      <span ref={highlightRef} className="proto-picker-highlight" aria-hidden="true" />
      {variants.map((name, i) => (
        <button
          key={name}
          ref={(el) => {
            itemRefs.current[i] = el;
          }}
          type="button"
          className="proto-picker-item"
          data-active={i === active ? "" : undefined}
          aria-current={i === active ? "true" : undefined}
          onClick={() => onChange(i)}
        >
          {name}
        </button>
      ))}
    </nav>
  );
}
