import { Unkey } from "@unkey/api";
import { NextRequest, NextResponse } from "next/server";

export async function GET(req: NextRequest) {
  const unkeyToken = process.env.UNKEY_TOKEN;
  if (!unkeyToken) {
    return new NextResponse("UNKEY_TOKEN is undefined", { status: 500 });
  }
  const unkey = new Unkey({ token: unkeyToken });
  // This part is a bit ugly, but we just use it here to make it easy to access this example
  // in real world code, you usually have the token in a header, where it's much easier to access
  const key = new URL(req.url).searchParams.get("key");
  if (!key) {
    return NextResponse.json({ error: "No key provided" }, { status: 400 });
  }

  const { result, error } = await unkey.keys.verify({ key });

  if (error) {
    return NextResponse.json({ error }, { status: 500 });
  }

  if (!result.valid) {
    // Do not grant access to your user!
    return NextResponse.json(result, { status: 400 });
  }

  // process the request here

  return new NextResponse(`Your API key is valid!

Here's the key that was generated for you: ${key}

And some more information from the verification:
${JSON.stringify(result, null, 2)}

  `);
}
