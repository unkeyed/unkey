"use server";

import { revalidatePath } from "next/cache";

async function revalidate() {
  await revalidatePath("/", "layout");
}
