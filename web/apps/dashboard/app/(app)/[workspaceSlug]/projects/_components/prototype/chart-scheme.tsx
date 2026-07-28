"use client";

import { useEffect } from "react";
import { type ChartScheme, useChartScheme } from "./scenario";

const ATTRIBUTE = "data-chart";

/**
 * Applies the prototype chart scheme as `data-chart` on <html>.
 *
 * Chart colours in this app are all `hsl(var(--token))` resolved at paint time,
 * so swapping the token values at the root recolours every chart — including the
 * real api and ratelimit pages, which know nothing about the prototype. The
 * scheme sets purpose-named `--chart-*` variables rather than overriding shared
 * steps like `--accent-4`, which also paint borders and backgrounds.
 *
 * The default scheme lives in `:root`, so charts paint correctly before this
 * effect runs; the attribute only ever switches to a non-default scheme.
 */
export function PrototypeChartScheme() {
  const { chartScheme } = useChartScheme();

  useEffect(() => {
    const root = document.documentElement;
    if (chartScheme === "semantic") {
      root.removeAttribute(ATTRIBUTE);
      return;
    }
    root.setAttribute(ATTRIBUTE, chartScheme satisfies ChartScheme);
    return () => root.removeAttribute(ATTRIBUTE);
  }, [chartScheme]);

  return null;
}
