import { NextResponse } from "next/server";

export async function POST(request: Request) {
  const { url } = await request.json();
  await fetch(url, {
    mode: "no-cors",
  });
  return NextResponse.json({ done: true });
}
