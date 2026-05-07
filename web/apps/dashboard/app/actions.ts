"use server";
import { getAuth } from "@/lib/auth/get-auth";
import { revalidatePath } from "next/cache";

export async function revalidate(path: string, segment?: "page" | "layout") {
  // Server Actions are publicly callable POST endpoints. Without this check,
  // any visitor who scrapes the action ID from a client bundle could trigger
  // revalidatePath on arbitrary paths, thrashing the Next.js Data and Router
  // caches.
  const { userId } = await getAuth();
  if (!userId) {
    throw new Error("Unauthorized");
  }
  revalidatePath(path, segment || "page");
}
