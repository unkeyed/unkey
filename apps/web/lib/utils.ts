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

  let cache = new Map<number, number>();
  let series: { value: number; time: number }[] = [];
  for (let i = startWindow; i < endWindow; i++) {
    let value = cache.get(i);
    if (!value) {
      value = data.find((d) => toStartOfInterval(d.time) === i)?.value ?? 0,
        cache.set(i, value);
    }
    series.push({
      time: i * granularity,
      value,
    });
  }
  return series

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
