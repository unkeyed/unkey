import { auth } from "@/auth";
import { verifyKey } from "@unkey/api";
import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import OpenAI from "openai";

import { readKeyFromCookie, resetCookieIfDeleted } from "./cookies";
// Create an OpenAI API client (that's edge friendly!)
const openai = new OpenAI({
  apiKey: process.env.OPENAI_API_KEY,
});

// Set the runtime to edge for best performance
export const runtime = "edge";

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
