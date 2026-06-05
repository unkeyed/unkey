"use client";

import { cn } from "@unkey/ui/src/lib/utils";
import { type CSSProperties, type ReactNode, useEffect, useRef } from "react";

// Renders a crisp foreground icon over a static, dotted "halftone echo" of the same shape:
// the icon's SVG is rasterized to an alpha mask, then sampled onto a 0/1 grid so its outline
// reads as a faint dotted contour behind it. Static by design — the live animated field is
// reserved for the single brand hero (see unkey-ascii.tsx).
const SRC = 256; // mask raster size
const SS = 2; // supersample per cell axis
const PAD = 0.12; // fraction of the field left as breathing room around the shape
const CUTOFF = 0.14; // alpha coverage below this is treated as empty space

type Props = {
  icon: ReactNode; // shape source — rasterized to the mask, never drawn on top
  cols?: number; // grid density — more columns = finer dots
  dotSize?: number; // px
  gap?: number; // px between dots
  className?: string;
};

export function IconHalftone({ icon, cols = 44, dotSize = 3, gap = 1.5, className }: Props) {
  const srcRef = useRef<HTMLSpanElement>(null);
  const preRef = useRef<HTMLPreElement>(null);

  useEffect(() => {
    const svg = srcRef.current?.querySelector("svg");
    const pre = preRef.current;
    if (!svg || !pre) {
      return;
    }

    // Clone the rendered icon and force opaque ink — we only read the alpha channel, so the
    // theme's currentColor (which resolves to transparent off-DOM) must become solid.
    const clone = svg.cloneNode(true) as SVGSVGElement;
    clone.setAttribute("width", String(SRC));
    clone.setAttribute("height", String(SRC));
    for (const el of Array.from(clone.querySelectorAll("*"))) {
      if (el.getAttribute("stroke") && el.getAttribute("stroke") !== "none") {
        el.setAttribute("stroke", "#000");
      }
      const fill = el.getAttribute("fill");
      if (fill && fill !== "none") {
        el.setAttribute("fill", "#000");
      }
    }
    const xml = new XMLSerializer().serializeToString(clone);

    const mc = document.createElement("canvas");
    mc.width = SRC;
    mc.height = SRC;
    const ctx = mc.getContext("2d", { willReadFrequently: true });
    const measEl = document.createElement("canvas").getContext("2d");
    if (!ctx || !measEl) {
      return;
    }

    const img = new Image();
    img.onload = () => {
      const inset = SRC * PAD;
      const box = SRC - inset * 2;
      ctx.clearRect(0, 0, SRC, SRC);
      ctx.drawImage(img, inset, inset, box, box);
      const alpha = ctx.getImageData(0, 0, SRC, SRC).data;

      measEl.font = `${dotSize}px ui-monospace, monospace`;
      const wc = measEl.measureText("0").width + gap;
      const hc = dotSize + gap;
      const rows = Math.max(4, Math.round((cols * wc) / hc));
      pre.style.fontSize = `${dotSize}px`;
      pre.style.lineHeight = `${hc}px`;
      pre.style.letterSpacing = `${gap}px`;

      const at = (u: number, v: number) => {
        const x = Math.min(SRC - 1, Math.max(0, (u * SRC) | 0));
        const y = Math.min(SRC - 1, Math.max(0, (v * SRC) | 0));
        return alpha[(y * SRC + x) * 4 + 3] / 255;
      };

      let out = "";
      for (let r = 0; r < rows; r++) {
        for (let c = 0; c < cols; c++) {
          let acc = 0;
          for (let sy = 0; sy < SS; sy++) {
            for (let sx = 0; sx < SS; sx++) {
              acc += at((c + (sx + 0.5) / SS) / cols, (r + (sy + 0.5) / SS) / rows);
            }
          }
          const cov = acc / (SS * SS);
          out += cov > CUTOFF ? (Math.random() < 0.5 ? "1" : "0") : " ";
        }
        if (r < rows - 1) {
          out += "\n";
        }
      }
      pre.textContent = out;
    };
    img.src = `data:image/svg+xml,${encodeURIComponent(xml)}`;
  }, [icon, cols, dotSize, gap]);

  const preStyle: CSSProperties = {
    margin: 0,
    whiteSpace: "pre",
    userSelect: "none",
    fontFamily: "var(--font-mono, ui-monospace, monospace)",
  };

  return (
    <div className={cn("relative inline-flex items-center justify-center", className)} aria-hidden>
      {/* hidden source: the real icon, read once to build the mask */}
      <span ref={srcRef} className="sr-only">
        {icon}
      </span>
      <pre ref={preRef} className="text-gray-10" style={preStyle} />
    </div>
  );
}
