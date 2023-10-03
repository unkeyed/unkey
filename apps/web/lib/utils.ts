import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
export function fillRange(
  data: { value: number; time: number }[],
  start: number,
  end: number,
): { value: number; time: number }[] {
  const t = new Date(start);
  const series: { value: number; time: number }[] = [];

  function toDay(unixmilli: number): string {
    const d = new Date(unixmilli);
    d.setUTCHours(0, 0, 0, 0);
    return d.toUTCString();
  }

  const lookup = data.reduce((acc, d) => {
    acc[toDay(d.time)] = d.value;
    return acc;
  }, {} as Record<string, number>);

  while (t.getTime() <= end) {
    const d = lookup[toDay(t.getTime())];
    series.push({
      time: t.getTime(),
      value: d ?? 0,
    });

    // Now increment the time
    t.setUTCDate(t.getUTCDate() + 1);
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