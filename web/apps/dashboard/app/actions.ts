"use server";
import { revalidatePath } from "next/cache";

export async function revalidate(path: string, segment?: "page" | "layout") {
  revalidatePath(path, segment || "page");
}


