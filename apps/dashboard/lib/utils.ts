import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
export const isBrowser = typeof window !== "undefined";

export function debounce<T extends (...args: any[]) => any>(
  func: T,
  delay: number
) {
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
