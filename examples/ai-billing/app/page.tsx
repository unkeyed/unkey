import { auth } from "@/auth";
import { Unkey } from "@unkey/api";
import type { Metadata } from "next";
import Link from "next/link";
import { ImageGenerator } from "../components/image-generator";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { IdCardIcon } from "@radix-ui/react-icons";

export const metadata: Metadata = {
  metadataBase: new URL("https://unkey.dev"),
  title: "AI billing example with Unkey",
  description:
    "Simple AI image generation application. Contains example code of generating and refilling Unkey API keys in response to a Stripe payment link, and using the `remaining` field for measuring usage.",
  openGraph: {
    title: "AI billing example with Unkey",
    images: ["https://unkey.dev/images/templates/unkey-stripe.png"],
  },
};

export default async function GeneratePage() {
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
    <div className="flex flex-col items-center">
      <div className="flex flex-col w-full max-w-2xl">
        {remainingByOwner === 0 && (
          <Alert variant="destructive" className="mb-8 border bg-yellow-100">
            <IdCardIcon className="h-4 w-4" />
            <AlertTitle>No credits remaining</AlertTitle>
            <AlertDescription>
              Please{" "}
              <Link href="/credits" className="underline">
                add credits
              </Link>{" "}
              in order to continue.
            </AlertDescription>
          </Alert>
        )}
        <div className="flex flex-col md:flex-row items-center w-full max-w-2xl justify-between">
          <h1 className="text-3xl text-center font-medium">Dall-E 3 Image Generation</h1>
          <div className="mt-4 md:mt-2 flex items-center gap-2">
            <span className="font-medium text-gray-900">Your credits:</span>
            <span className="font-semibold text-gray-900 text-xl">{remainingByOwner}</span>
          </div>
        </div>
      </div>
      <div>
        <ImageGenerator credits={remainingByOwner} />
      </div>
    </div>
  );
}
