import * as flags from "@/lib/flags";
import { getProviderData } from "@flags-sdk/vercel";
import { createFlagsDiscoveryEndpoint } from "flags/next";

export const GET = createFlagsDiscoveryEndpoint(async () => {
  return await getProviderData(flags);
});
