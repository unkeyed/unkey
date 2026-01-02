import type { NextRequest } from "next/server";
import { updateSession } from "./sessions";

export async function authMiddleware(request: NextRequest) {
  return await updateSession(request);
}
