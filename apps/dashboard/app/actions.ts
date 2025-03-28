"use server";
import { revalidatePath, revalidateTag } from "next/cache";

export async function revalidate(path: string, segment?: "page" | "layout") {
  revalidatePath(path, segment || "page");
}

export async function revalidateMyTag(slug: string) {
  revalidateTag(slug);
}

export { revalidateMyTag as revalidateTag };
