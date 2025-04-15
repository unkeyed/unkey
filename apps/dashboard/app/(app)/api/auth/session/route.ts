import { getAuth } from "@/lib/auth/get-auth";

export const dynamic = "force-dynamic";
export async function GET() {
  try {
    const { userId } = await getAuth();

    if (!userId) {
      return new Response(null, { status: 401 });
    }

    // Just return 200 OK if authenticated
    return new Response(null, { status: 200 });
  } catch (error) {
    console.error("Error checking session:", error);
    return new Response(null, { status: 401 });
  }
}
