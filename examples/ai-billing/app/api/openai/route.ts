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

function readKeyFromCookie(cookieName: string) {
  const cookieStore = cookies();
  const cookie = cookieStore.get(cookieName)?.value as string;
  if (!cookie) {
    return null;
  }
  return cookie;
}

async function resetCookieIfDeleted({ ownerId }: { ownerId: string }) {
  const unkey = new Unkey({ rootKey: process.env.UNKEY_ROOT_KEY! });

  const { error, result } = await unkey.apis.listKeys({
    apiId: process.env.UNKEY_API_ID!,
    ownerId,
  });

  if (error) {
    throw new Error("Error reading from Unkey: check your API ID");
  }

  const keys = result?.keys;

  if (keys.length) {
    const key = keys[0];
    const remaining = key.remaining;

    const newKey = await unkey.keys.create({
      apiId: process.env.UNKEY_API_ID!,
      ownerId,
      remaining,
    });

    if (newKey.error) {
      throw new Error("Error creating new key");
    }

    const newCookie = { key: newKey.result?.key, keyId: newKey.result?.keyId };

    await unkey.keys.delete({
      keyId: key.id,
    });

    return newCookie;
  } else {
    return null;
  }
}

export async function POST(req: Request, _res: Response) {
  const sess = await auth();
  const ownerId = sess?.user?.id;

  if (!ownerId) {
    return new Response("Unauthorized: please supply an owner ID", { status: 401 });
  }

  const cookie = readKeyFromCookie("unkey");
  let cookieObj;

  if (!cookie) {
    cookieObj = await resetCookieIfDeleted({ ownerId });
    const cookieStore = cookies();
    cookieStore.set("unkey", JSON.stringify(cookieObj));
  } else {
    cookieObj = JSON.parse(cookie);
  }

  const { key } = cookieObj;

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
