"use server";
import { revalidatePath, revalidateTag } from "next/cache";

export async function revalidate(path: string) {
  revalidatePath(path, "page");
}

export async function revalidateMyTag(slug: string) {
  revalidateTag(slug);
}

export { revalidateMyTag as revalidateTag };
