import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

import type { TRPCClientError } from "@trpc/client";
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
export const isBrowser = typeof window !== "undefined";

export function handleError(error: string): string {
  let message = JSON.parse(error);
  message = message.at(0).message;
  return message;
}
