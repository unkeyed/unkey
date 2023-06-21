import { NextRequest, NextResponse } from "next/server";
import { Unkey } from "@unkey/api";

const unkey = new Unkey({ token: process.env.UNKEY_TOKEN! });

export async function GET(req: NextRequest) {
  // This part is a bit ugly, but we just use it here to make it easy to access this example
  // in real world code, you usually have the token in a header, where it's much easier to access
  const key = new URL(req.url).searchParams.get("key");
  if (!key) {
    return NextResponse.json({ error: "No key provided" }, { status: 400 });
  }

  const verification = await unkey.keys.verify({ key });

  if (!verification.valid) {
    // Do not grant access to your user!
    return NextResponse.json(verification, { status: 400 });
  }

  // process the request here

  return new NextResponse(`Your API key is valid!

Here's the key that was generated for you: ${key}

And some more information from the verification:
${JSON.stringify(verification, null, 2)}
  
  `);
}
