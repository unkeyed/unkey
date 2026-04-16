import { useState } from "react";

interface Props {
  scale: string;
  alpha?: boolean;
}

const STEPS = Array.from({ length: 12 }, (_, i) => i + 1);

export function ColorSwatch({ scale, alpha = false }: Props) {
  return (
    <div className="mt-10 mb-4 grid grid-cols-6 md:grid-cols-12 gap-3 rounded-lg bg-background border border-grayA-3 pt-10 pb-14 px-8">
      {STEPS.map((step) => (
        <SwatchCell key={step} scale={scale} alpha={alpha} step={step} />
      ))}
    </div>
  );
}

function SwatchCell({
  scale,
  alpha,
  step,
}: {
  scale: string;
  alpha: boolean;
  step: number;
}) {
  const token = `${scale}-${step}`;
  const cssFn = alpha ? "hsla" : "hsl";
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(token);
      setCopied(true);
      setTimeout(() => setCopied(false), 1200);
    } catch {
      /* ignored */
    }
  }

  return (
    <button
      type="button"
      onClick={copy}
      aria-label={token}
      className="group relative aspect-[1/3] rounded-md cursor-pointer"
      style={{ background: `${cssFn}(var(--${token}))` }}
    >
      <span className="absolute left-1/2 -translate-x-1/2 -bottom-6 font-mono text-[10px] text-gray-10 pointer-events-none">
        {step}
      </span>
      <span
        className={`absolute left-1/2 -translate-x-1/2 bottom-full mb-2 px-2 py-1 rounded-[4px] bg-gray-12 text-gray-1 font-mono text-[10px] whitespace-nowrap transition-opacity duration-150 pointer-events-none z-10 ${
          copied ? "opacity-0" : "opacity-0 group-hover:opacity-100"
        }`}
      >
        {token}
      </span>
      <span
        className={`absolute inset-0 flex items-center justify-center rounded-md bg-gray-12 text-gray-1 text-[10px] font-medium uppercase tracking-[0.24em] transition-opacity pointer-events-none ${
          copied ? "opacity-100" : "opacity-0"
        }`}
      >
        Copied
      </span>
    </button>
  );
}
