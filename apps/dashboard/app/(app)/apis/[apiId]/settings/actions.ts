"use server";

import { revalidatePath, revalidateTag } from "next/cache";

export async function revalidate() {
  await revalidatePath("/", "layout");
}

export async function revalidateMyTag(tag: string) {
  revalidateTag(tag);
}
