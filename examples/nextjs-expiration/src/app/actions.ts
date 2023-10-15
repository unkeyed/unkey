"use server";

import { createApiKey } from "@/server/unkey-client";
import { revalidatePath } from "next/cache";

export async function createKey(_formDate: FormData) {
  await createApiKey({
    expires: new Date().getTime() + 1000 * 60,
  });

  revalidatePath("/");
}
