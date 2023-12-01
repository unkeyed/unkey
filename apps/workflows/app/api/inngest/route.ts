import { inngest } from "@/lib/inngest";
import { helloWorld } from "@/lib/workflows/hello-world";
import { updateUsage } from "@/lib/workflows/update-usage";
import { serve } from "inngest/next";

export const maxDuration = 300;

export const { GET, POST, PUT } = serve({
  client: inngest,
  functions: [helloWorld, updateUsage],
});
