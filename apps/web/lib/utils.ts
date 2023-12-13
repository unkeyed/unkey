import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
export function fillRange(
  data: { value: number; time: number }[],
  start: number,
  end: number,
  granularity: number,
): { value: number; time: number }[] {
  function toStartOfInterval(unixmilli: number): number {
    return Math.floor(unixmilli / granularity);
  }
  const startWindow = toStartOfInterval(start);
  const endWindow = toStartOfInterval(end);

  const cache = new Map<number, number>();
  for (const d of data) {
    cache.set(toStartOfInterval(d.time), d.value);
  }
  const series: { value: number; time: number }[] = [];
  for (let i = startWindow; i < endWindow; i++) {
    series.push({
      time: i * granularity,
      value: cache.get(i) ?? 0,
    });
  }
  return series;
}

export function cumulative(
  data: { value: number; time: number }[],
): { value: number; time: number }[] {
  let value = 0;
  return data.map((d) => {
    value += d.value;
    return {
      time: d.time,
      value,
    };
  });
}

export const isBrowser = typeof window !== "undefined";
