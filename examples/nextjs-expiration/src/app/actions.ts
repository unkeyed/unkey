"use server";

import { revalidatePath } from "next/cache";
import { createApiKey } from "@/server/unkey-client";

export async function createKey(formDate: FormData) {
  await createApiKey({
    expires: new Date().getTime() + 1000 * 60,
  });

  revalidatePath("/");
}
