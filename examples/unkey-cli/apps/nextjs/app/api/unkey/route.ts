import { Unkey } from "@unkey/api";
import { NextResponse } from "next/server";

export async function POST(request: Request) {
  const { id, redirect, code } = await request.json();
  if (!process.env.UNKEY_ROOT_KEY || !process.env.UNKEY_API_ID) {
    return NextResponse.json({
      statusCode: 500,
      message: "Unkey root key and API ID must be provided.",
    });
  }
  const unkey = new Unkey({ rootKey: process.env.UNKEY_ROOT_KEY });

  const { result, error } = await unkey.keys.create({
    apiId: process.env.UNKEY_API_ID,
    prefix: "cli_demo",
    ownerId: id,
  });

  if (error) {
    return NextResponse.json({
      statusCode: 500,
      message: "Error creating key – please ensure apiId is valid.",
    });
  }

  return NextResponse.json({ ...result, code, redirect });
}
