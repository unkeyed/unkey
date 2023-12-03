import { inngest } from "@/lib/inngest";
import { createInvoice } from "@/lib/workflows/create-invoice";
import { invoicing } from "@/lib/workflows/invoicing";
import { serve } from "inngest/next";

export const maxDuration = 300;

export const { GET, POST, PUT } = serve({
  client: inngest,
  functions: [invoicing, createInvoice],
});
