import { auth } from "@/lib/auth/server";
import { setCookie } from "@/lib/auth/cookies";
import { UNKEY_SESSION_COOKIE } from "@/lib/auth/types";

export async function POST(request: Request) {
  try {
    // Get the current token from the request
    const currentToken = request.headers.get('x-current-token');
    
    // Call refreshSession logic here and get new token
    const { newToken, expiresAt } = await auth.sessionRefresh(currentToken);
    
    // Set the new cookie using your utility
    await setCookie({
      name: UNKEY_SESSION_COOKIE,
      value: newToken,
      options: {
        httpOnly: true,
        secure: true,
        sameSite: "lax",
        path: '/',
        maxAge: Math.floor((expiresAt.getTime() - Date.now()) / 1000) // Convert to seconds
      }
    });

    return Response.json({ success: true });
  } catch (error) {
    console.error("Session refresh failed:", error);
    return Response.json(
      { success: false, error: "Failed to refresh session" }, 
      { status: 401 }
    );
  }
}