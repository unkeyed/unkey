import { verifyKey } from "@unkey/api";
import { NextResponse } from "next/server";
export async function GET(request: Request) {
    const header = request.headers.get('Authorization');
    if (!header) {
        return new Response('No Authorization header', { status: 401 });
    }
    const token = header.replace('Bearer ', '');
    const { result, error } = await verifyKey(token);

    if (error) {
        console.error(error.message);
        return new Response("Internal Server Error", { status: 500 });
    }

    if (!result.valid) {
        // do not grant access
        return new Response('Unauthorized', { status: 401 });
    }

    // process request
    return NextResponse.json({result})
}