import { stripeEnv } from "@/lib/env";
import { ratelimit, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { mapProduct } from "../utils/stripe";
import { z } from "zod";

const productSchema = z.object({
  id: z.string(),
  name: z.string(),
  priceId: z.string(),
  dollar: z.number(),
  quotas: z.object({
    requestsPerMonth: z.number(),
  }),
});

export const getProducts = t.procedure
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .output(z.array(productSchema))
  .query(async ({ ctx }) => {
    const e = stripeEnv();
    if (!e) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Stripe is not configured",
      });
    }

    const stripe = new Stripe(e.STRIPE_SECRET_KEY, {
      apiVersion: "2023-10-16",
      typescript: true,
    });

    const products = await stripe.products
      .list({
        active: true,
        ids: e.STRIPE_PRODUCT_IDS_PRO,
        limit: 100,
        expand: ["data.default_price"],
      })
      .then((res) =>
        res.data.map(mapProduct).sort((a, b) => a.dollar - b.dollar)
      );

    return products;
  });
