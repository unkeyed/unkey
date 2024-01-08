import { revalidate } from "@/app/revalidate";
import { NextRequest, NextResponse } from "next/server";
import Stripe from "stripe";
import { createKey, listKeys, updateKey } from "./keys";

export async function GET(request: NextRequest, _response: Response) {
  const url = new URL(request.url);
  const session_id = url.searchParams.get("session_id") as string;

  if (!session_id) {
    return new Response("No session_id", { status: 400 });
  }

  const stripe = new Stripe(process.env.STRIPE_SECRET_KEY!, {
    typescript: true,
  });

  const session = await stripe.checkout.sessions.retrieve(session_id);

  if (!session) {
    return new Response("No session", { status: 400 });
  }

  const keys = await listKeys();
  if (!keys.length) {
    const { key, keyId } = await createKey();
    const data = { key, keyId };
    const response = NextResponse.redirect(new URL("/credits", request.url));
    response.cookies.set({
      name: "unkey",
      value: JSON.stringify(data),
      httpOnly: true,
    });
    revalidate("/credits");
    return response;
  } else {
    const currentKey = keys[0];
    await updateKey(currentKey);
    revalidate("/credits");
    const response = NextResponse.redirect(new URL("/credits", request.url));
    return response;
  }
}
