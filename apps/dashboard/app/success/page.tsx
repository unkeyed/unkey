import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { Empty } from "@unkey/ui";
import { redirect } from "next/navigation";
import Stripe from "stripe";
import { SuccessClient } from "./client";

type Props = {
  searchParams: {
    session_id?: string;
  };
};

export default async function SuccessPage(props: Props) {
  // If no session_id, just redirect back to billing
  // This will make a user login if they are not logged in
  // This will also redirect to the billing page if the user is logged in
  if (!props.searchParams.session_id) {
    return <SuccessClient />;
  }

  // Process the Stripe session and update workspace
  const e = stripeEnv();
  if (!e) {
    return (
      <Empty>
        <Empty.Title>Stripe is not configured</Empty.Title>
        <Empty.Description>
          If you are selfhosting Unkey, you need to configure Stripe in your environment variables.
        </Empty.Description>
      </Empty>
    );
  }

  const stripe = new Stripe(e.STRIPE_SECRET_KEY, {
    apiVersion: "2023-10-16",
    typescript: true,
  });

  try {
    const session = await stripe.checkout.sessions.retrieve(props.searchParams.session_id);

    if (!session) {
      console.warn("Stripe session not found");
      return redirect("/auth/sign-in");
    }

    const workspaceReference = session.client_reference_id;
    if (!workspaceReference) {
      console.warn("Stripe session client_reference_id not found");
      return <SuccessClient />;
    }

    const ws = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.id, workspaceReference), isNull(table.deletedAtM)),
    });

    if (!ws) {
      console.warn("Workspace not found");
      redirect("/new");
    }

    const customer = await stripe.customers.retrieve(session.customer as string);
    if (!customer || !session.setup_intent) {
      console.warn("Stripe customer not found");
      return <SuccessClient />;
    }

    const setupIntent = await stripe.setupIntents.retrieve(session.setup_intent.toString());

    if (!setupIntent.payment_method) {
      console.warn("Stripe payment method not found");
      return <SuccessClient />;
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
    } catch (error) {
      console.error("Failed to update workspace:", error);
      return (
        <Empty>
          <Empty.Title>Failed to update workspace</Empty.Title>
          <Empty.Description>
            There was an error updating your workspace with payment information. Please contact
            support@unkey.dev.
          </Empty.Description>
        </Empty>
      );
    }
  } catch (error) {
    console.error("Error processing Stripe session:", error);
    return (
      <Empty>
        <Empty.Title>Failed to update workspace</Empty.Title>
        <Empty.Description>
          There was an error updating your workspace with payment information. Please contact
          support@unkey.dev.
        </Empty.Description>
      </Empty>
    );
  }

  return <SuccessClient />;
}
