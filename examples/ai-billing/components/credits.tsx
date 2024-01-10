import { auth } from "@/auth";
import { Unkey } from "@unkey/api";

export async function Credits() {
  const sess = await auth();
  const ownerId = sess?.user?.id;
  const unkey = new Unkey({ rootKey: process.env.UNKEY_ROOT_KEY! });

  const allKeys = await unkey.apis.listKeys({
    apiId: process.env.UNKEY_API_ID!,
    ownerId,
  });

  if (allKeys.error) {
    return <div>Error retrieving credits</div>;
  }

  const remainingByOwner = allKeys.result.keys.reduce((acc, key) => acc + (key.remaining || 0), 0);

  return (
    <p className="font-medium text-gray-600">
      Your credits: <span className="font-semibold text-xl">{remainingByOwner}</span>
    </p>
  );
}
