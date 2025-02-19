import type { TimeUnit } from "@unkey/ui";
import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
export const isBrowser = typeof window !== "undefined";

export function debounce<T extends (...args: any[]) => any>(func: T, delay: number) {
  let timeoutId: ReturnType<typeof setTimeout>;

  function debounced(...args: Parameters<T>) {
    clearTimeout(timeoutId);
    timeoutId = setTimeout(() => {
      func(...args);
    }, delay);
  }

  debounced.cancel = () => {
    clearTimeout(timeoutId);
  };

  return debounced;
}

type ThrottleOptions = {
  leading?: boolean; // Whether to invoke on the leading edge
  trailing?: boolean; // Whether to invoke on the trailing edge
};

type Timer = ReturnType<typeof setTimeout>;

export function throttle<T extends (...args: any[]) => any>(
  func: T,
  wait: number,
  options: ThrottleOptions = {},
): {
  (this: ThisParameterType<T>, ...args: Parameters<T>): ReturnType<T> | undefined;
  cancel: () => void;
  flush: () => ReturnType<T> | undefined;
} {
  let timeout: Timer | undefined;
  let result: ReturnType<T> | undefined;
  let previous = 0;
  let pending = false;

  const { leading = true, trailing = true } = options;

  // Function to handle the actual invocation
  function invokeFunc(time: number, args: Parameters<T>): ReturnType<T> {
    previous = leading ? time : 0;
    timeout = undefined;
    result = func.apply(null, args);
    pending = false;
    return result as ReturnType<T>;
  }

  // Function to handle the trailing edge call
  function trailingEdge(time: number, args: Parameters<T>): ReturnType<T> | undefined {
    timeout = undefined;

    if (trailing && pending) {
      return invokeFunc(time, args);
    }
    pending = false;
    return result;
  }

  // Function to schedule the delayed invocation
  function remainingWait(time: number) {
    const timeSinceLastCall = time - previous;
    const timeWaiting = wait - timeSinceLastCall;
    return timeWaiting;
  }

  // The actual throttled function
  function throttled(
    this: ThisParameterType<T>,
    ...args: Parameters<T>
  ): ReturnType<T> | undefined {
    const time = Date.now();
    const isInvoking = shouldInvoke(time);

    pending = true;

    if (isInvoking) {
      if (timeout) {
        clearTimeout(timeout);
        timeout = undefined;
      }
      return invokeFunc(time, args);
    }

    if (!timeout && trailing) {
      timeout = setTimeout(() => trailingEdge(Date.now(), args), remainingWait(time));
    }

    return result;
  }

  // Helper to determine if we should invoke the function
  function shouldInvoke(time: number): boolean {
    const timeSinceLastCall = time - previous;
    return (
      (previous === 0 && leading) || // First call with leading edge
      timeSinceLastCall >= wait // Enough time has passed
    );
  }

  // Cancel method
  throttled.cancel = (): void => {
    if (timeout) {
      clearTimeout(timeout);
    }
    previous = 0;
    timeout = undefined;
    result = undefined;
    pending = false;
  };

  // Flush method
  throttled.flush = (): ReturnType<T> | undefined => {
    if (timeout) {
      return trailingEdge(Date.now(), [] as unknown as Parameters<T>);
    }
    return result;
  };

  return throttled;
}

export const getTimestampFromRelative = (relativeTime: string): number => {
  if (!relativeTime.match(/^(\d+[whdm])+$/)) {
    throw new Error(
      'Invalid relative time format. Expected format: combination of numbers followed by w, h, d, or m (e.g., "1h", "2d", "30m", "1w", "1w2d")',
    );
  }
  let totalMilliseconds = 0;
  for (const [, amount, unit] of relativeTime.matchAll(/(\d+)([whdm])/g)) {
    const value = Number.parseInt(amount, 10);
    switch (unit) {
      case "w":
        totalMilliseconds += value * 7 * 24 * 60 * 60 * 1000;
        break;
      case "h":
        totalMilliseconds += value * 60 * 60 * 1000;
        break;
      case "d":
        totalMilliseconds += value * 24 * 60 * 60 * 1000;
        break;
      case "m":
        totalMilliseconds += value * 60 * 1000;
        break;
    }
  }
  return Date.now() - totalMilliseconds;
};

export const processTimeFilters = (date?: Date, newTime?: TimeUnit) => {
  if (date) {
    const hours = newTime?.HH ? Number.parseInt(newTime.HH) : 0;
    const minutes = newTime?.mm ? Number.parseInt(newTime.mm) : 0;
    const seconds = newTime?.ss ? Number.parseInt(newTime.ss) : 0;
    date.setHours(hours, minutes, seconds, 0);
    return date;
  }
  const now = new Date();
  return now;
};
