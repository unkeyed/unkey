import { workosAuthEnv } from "@/lib/env";
import { cache } from "react";

export const getWorkOSSession = cache(async () => {
  workosAuthEnv();
  const { withAuth } = await import("@workos-inc/authkit-nextjs");
  return withAuth();
});
