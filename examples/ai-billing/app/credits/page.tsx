import { auth } from "@/auth";
import { Card } from "@tremor/react";
import { Unkey } from "@unkey/api";
import { SelectCredits } from "./select";

export default async function CreditsPage() {
  const sess = await auth();

  const ownerId = sess?.user?.id;

  const unkey = new Unkey({ rootKey: process.env.UNKEY_ROOT_KEY! });

  const allKeys = await unkey.apis.listKeys({
    apiId: process.env.UNKEY_API_ID!,
    ownerId,
  });

  if (allKeys.error) {
    return <div>{allKeys.error.message}</div>;
  }

  const remainingByOwner = allKeys.result.keys.reduce((acc, key) => acc + (key.remaining || 0), 0);

  return (
    <div className="flex flex-col items-center">
      <div className="flex flex-col md:flex-row items-center w-full max-w-2xl justify-between">
        <h1 className="text-3xl text-center font-medium">Dall-E 3 Image Generation</h1>
        <div className="mt-4 md:mt-2 flex items-center gap-2">
          <span className="font-medium text-gray-900">Your credits:</span>
          <span className="font-semibold text-gray-900 text-xl">{remainingByOwner}</span>
        </div>
      </div>
      <Card className="max-w-[320px] md:max-w-2xl mt-10">
        <h1 className="text-3xl font-medium">Purchase API credits</h1>
        <p className="text-gray-600 mt-2">
          Please select the number of credits you want to purchase.
        </p>
        <p className="text-gray-600 mt-2">
          For demo purposes, enter '4242 4242 4242 4242' as your credit card, and any data for the
          other fields. This will not charge your card.
        </p>
        <SelectCredits />
      </Card>
    </div>
  );
}
