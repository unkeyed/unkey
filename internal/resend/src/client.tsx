import { Resend as Client } from "resend";

import { render } from "@react-email/render";
import React from "react";
import { PaymentIssue } from "../emails/payment_issue";
import { SubscriptionEnded } from "../emails/subscription_ended";
import { TrialEnded } from "../emails/trial_ended";
import UsageBudgetExceeded from "../emails/usage-budget-exceeded";
import { WelcomeEmail } from "../emails/welcome_email";

export class Resend {
  public readonly client: Client;
  private readonly replyTo = "support@unkey.dev";

  constructor(opts: { apiKey: string }) {
    this.client = new Client(opts.apiKey);
  }

  public async sendTrialEnded(req: {
    email: string;
    name: string;
    workspace: string;
  }): Promise<void> {
    const html = render(<TrialEnded username={req.name} workspaceName={req.workspace} />);
    try {
      const result = await this.client.emails.send({
        to: req.email,
        from: "james@updates.unkey.dev",
        reply_to: this.replyTo,
        subject: "Your Unkey trial has ended",
        html,
      });

      if (!result.error) {
        return;
      }
      throw result.error;
    } catch (error) {
      console.error("Error occurred sending subscription email ", JSON.stringify(error));
    }
  }

  public async sendSubscriptionEnded(req: {
    email: string;
    name: string;
  }): Promise<void> {
    const html = render(<SubscriptionEnded username={req.name} />);
    try {
      const result = await this.client.emails.send({
        to: req.email,
        from: "james@updates.unkey.dev",
        reply_to: this.replyTo,
        subject: "Your Unkey trial has ended",
        html,
      });
      if (!result.error) {
        return;
      }
      throw result.error;
    } catch (error) {
      console.error("Error occurred sending subscription email ", JSON.stringify(error));
    }
  }

  public async sendWelcomeEmail(req: { email: string }) {
    const html = render(<WelcomeEmail />);
    try {
      const result = await this.client.emails.send({
        to: req.email,
        from: "james@updates.unkey.dev",
        reply_to: this.replyTo,
        subject: "Welcome to Unkey",
        html,
      });
      if (!result.error) {
        return;
      }
      throw result.error;
    } catch (error) {
      console.error("Error occurred sending welcome email ", JSON.stringify(error));
    }
  }

  public async sendPaymentIssue(req: {
    email: string;
    name: string;
    date: Date;
  }): Promise<void> {
    const html = render(<PaymentIssue username={req.name} date={req.date.toDateString()} />);
    try {
      const result = await this.client.emails.send({
        to: req.email,
        from: "james@updates.unkey.dev",
        reply_to: this.replyTo,
        subject: "There was an issue with your payment",
        html,
      });
      if (!result.error) {
        return;
      }
      throw result.error;
    } catch (error) {
      console.error("Error occurred sending payment issue email ", JSON.stringify(error));
    }
  }

  public async sendBatchBudgetExceeded(
    batch: {
      email: string;
      ccEmails: string[];
      workspaceName: string;
      budgetedAmount: number;
      currentPeriodBilling: number;
    }[],
  ): Promise<{ success: true; error?: null } | { success: false; error: string }> {
    try {
      if (batch.length > 100) {
        throw new Error("Allowed up to 100 batch emails.");
      }

      const result = await this.client.batch.send(
        batch.map((data) => {
          const html = render(
            <UsageBudgetExceeded
              workspace={data.workspaceName}
              budgetedAmount={data.budgetedAmount}
              currentPeriodBilling={data.currentPeriodBilling}
            />,
          );

          return {
            to: data.email,
            cc: data.ccEmails,
            from: "james@updates.appdoc.io",
            reply_to: this.replyTo,
            subject: "Budget Notification Usage Exceeded",
            html,
          };
        }),
      );

      if (!result.error) {
        return { success: true };
      }

      console.error(
        "Error occurred batch sending budget exceeded email ",
        JSON.stringify(result.error),
      );
      return { success: false, error: result.error.message };
    } catch (error) {
      console.error(
        "Error occurred batch sending budget exceeded email ",
        (error as Error).message,
      );
      return { success: false, error: (error as Error).message };
    }
  }
}
