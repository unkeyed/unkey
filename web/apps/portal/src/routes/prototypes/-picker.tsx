import { useEffect, useLayoutEffect, useRef, useState } from "react";

const CSS = `
.proto-picker{position:fixed;left:50%;transform:translateX(-50%);z-index:2147483647;display:flex;align-items:center;gap:2px;padding:4px;border-radius:999px;background:rgba(10,10,10,.82);-webkit-backdrop-filter:blur(12px) saturate(1.4);backdrop-filter:blur(12px) saturate(1.4);box-shadow:0 0 0 1px rgba(255,255,255,.08) inset,0 8px 24px rgba(0,0,0,.24),0 2px 6px rgba(0,0,0,.12);font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;font-size:13px;line-height:1;-webkit-font-smoothing:antialiased;user-select:none;-webkit-user-select:none}
.proto-picker-highlight{position:absolute;top:4px;left:0;height:28px;border-radius:999px;background:rgba(255,255,255,.12);will-change:transform}
.proto-picker[data-ready] .proto-picker-highlight{transition:transform 250ms cubic-bezier(.23,1,.32,1),width 250ms cubic-bezier(.23,1,.32,1)}
@media (prefers-reduced-motion:reduce){.proto-picker[data-ready] .proto-picker-highlight{transition:none}}
.proto-picker-item{position:relative;display:flex;align-items:center;height:28px;padding:0 12px;border:0;border-radius:999px;background:transparent;color:rgba(255,255,255,.55);font:inherit;cursor:pointer;transition:color 150ms ease-out}
.proto-picker-item:hover{color:rgba(255,255,255,.85)}
.proto-picker-item:active{transform:scale(.97)}
.proto-picker-item:focus-visible{outline:2px solid rgba(255,255,255,.4);outline-offset:2px}
.proto-picker-item[data-active]{color:#fff}
.proto-picker-label{color:rgba(255,255,255,.35);font-size:11px;padding:0 6px 0 10px;text-transform:uppercase;letter-spacing:.04em}
`;

type Props = {
  label: string;
  names: string[];
  index: number;
  onChange: (index: number) => void;
  bottom: number;
  keyboard?: boolean;
};

export function Picker({ label, names, index, onChange, bottom, keyboard = false }: Props) {
  const itemRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const [highlight, setHighlight] = useState({ width: 0, x: 0 });
  const [ready, setReady] = useState(false);

  useLayoutEffect(() => {
    const el = itemRefs.current[index];
    if (el) {
      setHighlight({ width: el.offsetWidth, x: el.offsetLeft });
    }
  }, [index]);

  useEffect(() => {
    const id = requestAnimationFrame(() => requestAnimationFrame(() => setReady(true)));
    return () => cancelAnimationFrame(id);
  }, []);

  useEffect(() => {
    if (!keyboard) {
      return;
    }
    function onKey(e: KeyboardEvent) {
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
      if (num >= 1 && num <= names.length) {
        onChange(num - 1);
      } else if (e.key === "ArrowRight") {
        onChange((index + 1) % names.length);
      } else if (e.key === "ArrowLeft") {
        onChange((index - 1 + names.length) % names.length);
      }
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [index, names.length, onChange, keyboard]);

  return (
    <>
      <style>{CSS}</style>
      <nav
        className="proto-picker"
        aria-label={`Prototype ${label}`}
        data-ready={ready ? "" : undefined}
        style={{ bottom }}
      >
        <span className="proto-picker-label">{label}</span>
        <span
          className="proto-picker-highlight"
          aria-hidden="true"
          style={{ width: highlight.width, transform: `translateX(${highlight.x}px)` }}
        />
        {names.map((name, i) => (
          <button
            key={name}
            type="button"
            ref={(el) => {
              itemRefs.current[i] = el;
            }}
            className="proto-picker-item"
            data-active={i === index ? "" : undefined}
            aria-current={i === index ? "true" : undefined}
            onClick={() => onChange(i)}
          >
            {name}
          </button>
        ))}
      </nav>
    </>
  );
}
