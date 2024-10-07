"use server";

import { revalidatePath, revalidateTag } from "next/cache";

async function revalidate() {
  await revalidatePath("/", "layout");
}

async function revalidateMyTag(tag: string) {
  revalidateTag(tag);
}

export { revalidate, revalidateMyTag as revalidateTag };
