import { ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 *
 */
export function fillRange(
  data: { value: number; time: number }[],
  start: number,
  end: number,
  step: "1m" | "1h" | "1d",
): { value: number; time: number }[] {
  const t = new Date(start);
  const series: { value: number; time: number }[] = [];
  while (t.getTime() <= end) {
    const d = data.find((u) => {
      switch (step) {
        case "1m":
          return new Date(u.time).setUTCSeconds(0, 0) === new Date(t).setUTCSeconds(0, 0);

        case "1h":
          return new Date(u.time).setUTCMinutes(0, 0, 0) === new Date(t).setUTCMinutes(0, 0, 0);
        case "1d":
          return new Date(u.time).setUTCHours(0, 0, 0, 0) === new Date(t).setUTCHours(0, 0, 0, 0);

        default:
          throw new Error(`Unhandled step: ${step}`);
      }
    });
    series.push({
      time: t.getTime(),
      value: d?.value ?? 0,
    });

    // Now increment the time
    switch (step) {
      case "1m":
        t.setUTCMinutes(t.getUTCMinutes() + 1);
        break;
      case "1h":
        t.setUTCHours(t.getUTCHours() + 1);
        break;
      case "1d":
        t.setUTCDate(t.getUTCDate() + 1);
        break;

      default:
        throw new Error(`Unhandled step: ${step}`);
    }
  }
  return series;
}
