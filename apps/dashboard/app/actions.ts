import { revalidatePath } from "next/cache";

export async function revalidate(path: string) {
  revalidatePath(path, "page");
}
