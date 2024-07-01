import { revalidateTag } from "next/cache";

export async function revalidate(path: string) {
  revalidateTag(path);
}
