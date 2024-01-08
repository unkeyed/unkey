import { auth } from "@/auth";
import { Unkey } from "@unkey/api";
import { redirect } from "next/navigation";
import Client from "./client";

export default async function KeysPage() {
  const sess = await auth();

  const ownerId = sess?.user?.id ?? sess?.user?.email;
  if (!ownerId) {
    return redirect("/sign-in");
  }

  const unkey = new Unkey({ rootKey: process.env.UNKEY_ROOT_KEY! });

  const { error, result } = await unkey.apis.listKeys({
    apiId: process.env.UNKEY_API_ID!,
    ownerId,
  });
  if (error) {
    console.error(error);
    return <div>{error.message}</div>;
  }

  return <Client keys={result?.keys ?? []} />;
}
