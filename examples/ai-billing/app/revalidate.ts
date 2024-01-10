"use server";

import { revalidateTag } from "next/cache";

export async function revalidate(path: string) {
  await revalidateTag(path);
}
