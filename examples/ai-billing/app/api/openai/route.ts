import { auth } from "@/auth";
import { Unkey, verifyKey } from "@unkey/api";
import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import OpenAI from "openai";

// Create an OpenAI API client (that's edge friendly!)
const openai = new OpenAI({
  apiKey: process.env.OPENAI_API_KEY,
});

// Set the runtime to edge for best performance
export const runtime = "edge";

export async function POST(req: Request, _res: Response) {
  const cookieStore = cookies();
  const sess = await auth();
  const ownerId = sess?.user?.id;

  const cookie = cookieStore.get("unkey")?.value as string;
  let cookieObj;

  if (!cookie) {
    const unkey = new Unkey({ rootKey: process.env.UNKEY_ROOT_KEY! });

    const { error, result } = await unkey.apis.listKeys({
      apiId: process.env.UNKEY_API_ID!,
      ownerId,
    });

    if (error) {
      return new Response("Error reading from Unkey – check your API ID", {
        status: 500,
      });
    }

    const keys = result?.keys;

    if (keys.length) {
      const key = keys[0];
      const remaining = key.remaining;

      console.log({ remaining });

      const newKey = await unkey.keys.create({
        apiId: process.env.UNKEY_API_ID!,
        ownerId,
        remaining,
      });

      if (newKey.error) {
        return new Response("Error creating new key", { status: 500 });
      }

      cookieObj = { key: newKey.result?.key, keyId: newKey.result?.keyId };

      await unkey.keys.delete({
        keyId: key.id,
      });

      cookieStore.set("unkey", JSON.stringify(cookieObj));
    } else {
      return new Response("Unauthorized", { status: 401 });
    }
  }

  // setting cookie is async, so we can read a cookie if already set, but not if we had to set it in if statement above
  if (!cookieObj) {
    cookieObj = JSON.parse(cookie);
  }

  const key = cookieObj?.key;

  if (!key) {
    return new Response("Unauthorized", { status: 401 });
  }

  const { result, error } = await verifyKey({
    key,
    apiId: process.env.UNKEY_API_ID!,
  });
  if (error) {
    return new Response(error.message, { status: 500 });
  }
  if (!result.valid) {
    return new Response("Unauthorized", { status: 401 });
  }
  const { prompt } = await req.json();

  const images = await Promise.all(
    new Array(2).fill(1).map(async () => {
      const image = await openai.images.generate({ model: "dall-e-3", prompt });
      return image.data.at(0)?.url;
    }),
  );

  return NextResponse.json({ images });
}
