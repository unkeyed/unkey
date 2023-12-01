import { inngest } from "@/lib/inngest";
import { updateUsage } from "@/lib/workflows/update-usage";
import { serve } from "inngest/next";

export const maxDuration = 300;

export const { GET, POST, PUT } = serve({
  client: inngest,
  functions: [updateUsage],
});
