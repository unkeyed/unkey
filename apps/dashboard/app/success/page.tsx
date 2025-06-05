import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { Empty } from "@unkey/ui";
import Stripe from "stripe";

import { redirect } from "next/navigation";

type Props = {
  searchParams: {
    session_id?: string;
  };
};

export default async function SuccessPage(props: Props) {
  // If no session_id, just show the success page without processing
  if (!props.searchParams.session_id) {
    return redirect("/auth/sign-in");
  }

  // Process the Stripe session and update workspace
  const e = stripeEnv();
  if (!e) {
    return (
      <Empty>
        <Empty.Title>Stripe is not configured</Empty.Title>
        <Empty.Description>
          If you are selfhosting Unkey, you need to configure Stripe in your
          environment variables.
        </Empty.Description>
      </Empty>
    );
  }

  const stripe = new Stripe(e.STRIPE_SECRET_KEY, {
    apiVersion: "2023-10-16",
    typescript: true,
  });

  try {
    const session = await stripe.checkout.sessions.retrieve(
      props.searchParams.session_id
    );

    if (!session) {
      console.warn("Stripe session not found");
      return redirect("/auth/sign-in");
    }

    const ws = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(
          eq(table.id, session.client_reference_id!!),
          isNull(table.deletedAtM)
        ),
    });

    if (!ws) {
      console.warn("Workspace not found");
      return redirect("/new");
    }

    const customer = await stripe.customers.retrieve(
      session.customer as string
    );
    if (!customer) {
      console.warn("Stripe customer not found");
      return redirect("settings/billing");
    }

    if (!session.setup_intent) {
      console.warn("Stripe setup intent not found");
      return redirect("settings/billing");
    }

    const setupIntent = await stripe.setupIntents.retrieve(
      session.setup_intent.toString()
    );

    if (!setupIntent.payment_method) {
      console.warn("Stripe payment method not found");
      return redirect("settings/billing");
    }

    // Update customer with default payment method
    await stripe.customers.update(customer.id, {
      invoice_settings: {
        default_payment_method: setupIntent.payment_method.toString(),
      },
    });

    // Update workspace with stripe customer ID
    try {
      await db
        .update(schema.workspaces)
        .set({
          stripeCustomerId: customer.id,
        })
        .where(eq(schema.workspaces.id, ws.id));

      console.log("Workspace updated with payment information");
    } catch (error) {
      console.error("Failed to update workspace:", error);
      return (
        <Empty>
          <Empty.Title>Failed to update workspace</Empty.Title>
          <Empty.Description>
            There was an error updating your workspace with payment information.
            Please contact support@unkey.dev.
          </Empty.Description>
        </Empty>
      );
    }
  } catch (error) {
    console.error("Error processing Stripe session:", error);
    // Still show success page even if there's an error
  }

  return redirect("/settings/billing");
}
