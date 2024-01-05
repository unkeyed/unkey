import { revalidate } from "@/app/revalidate";
import Stripe from "stripe";
import { Client } from "./client";
import { getCookie, setCookie } from "./cookies";

type Props = {
  searchParams: {
    session_id: string;
  };
};

export default async function StripeSuccessPage(props: Props) {
  const stripe = new Stripe(process.env.STRIPE_SECRET_KEY!, {
    typescript: true,
  });

  const session = await stripe.checkout.sessions.retrieve(props.searchParams.session_id);

  if (!session) {
    return <div>Stripe session not found</div>;
  }

  return (
    <>
      <Client setCookie={setCookie} getCookie={getCookie} revalidate={revalidate} />
      {/* <Server /> */}
    </>
  );
}
