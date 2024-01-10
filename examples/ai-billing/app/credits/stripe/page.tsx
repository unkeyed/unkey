import { auth } from "@/auth";
import { redirect } from "next/navigation";
import Stripe from "stripe";

export default async function StripePage({
  searchParams,
}: {
  searchParams: { value: string };
}) {
  const sess = await auth();

  const ownerId = sess?.user?.id;
  const ownerEmail = sess?.user?.email as string;

  const stripe = new Stripe(process.env.STRIPE_SECRET_KEY!, {
    typescript: true,
  });

  const baseUrl = process.env.VERCEL_URL
    ? `https:${process.env.VERCEL_URL}`
    : "http://localhost:3000";

  const successUrl = `${baseUrl}/credits/stripe/success?session_id={CHECKOUT_SESSION_ID}`;

  const session = await stripe.checkout.sessions.create({
    client_reference_id: ownerId,
    success_url: successUrl,
    mode: "payment",
    line_items: [
      {
        price: process.env.STRIPE_PRICE_ID,
        quantity: parseInt(searchParams.value) || 1,
      },
    ],
    currency: "USD",
    customer_creation: "always",
    customer_email: ownerEmail,
  });

  if (!session.url) {
    return <div>Could not create checkout session</div>;
  }

  return redirect(session.url);
}
