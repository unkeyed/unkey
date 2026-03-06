import type React from "react";

type SliderOption = { readonly label: string; readonly value: number };

export function valueToIndex<T extends readonly SliderOption[]>(options: T, value: number): number {
  const idx = options.findIndex((o) => o.value === value);
  return idx >= 0 ? idx : 0;
}

export function indexToValue<T extends readonly SliderOption[]>(
  options: T,
  index: number,
  fallback: number,
): number {
  return options[index]?.value ?? fallback;
}

export function buildSliderRangeStyle(
  currentIndex: number,
  maxIndex: number,
  minIndex: number,
  colorVar: string,
): React.CSSProperties {
  const range = maxIndex - minIndex;
  const normalized = range > 0 ? (currentIndex - minIndex) / range : 0;
  return {
    background: `linear-gradient(to right, hsla(var(--${colorVar}-4)), hsla(var(--${colorVar}-12)))`,
    backgroundSize: `${normalized > 0 ? 100 / normalized : 100}% 100%`,
    backgroundRepeat: "no-repeat",
  };
}
