import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
export const isBrowser = typeof window !== "undefined";

export function parseTrpcError(error: { message: string }): string {
  const messages = JSON.parse(error.message) as Array<{ message: string }>;
  return messages.at(0)?.message ?? "Unknown error, please contact support@unkey.dev";
}
