import { Resend as Client } from "resend";

import { render } from "@react-email/render";
import React from "react";
import { PaymentIssue } from "../emails/payment_issue";
import { SubscriptionEnded } from "../emails/subscription_ended";
import { TrialEnded } from "../emails/trial_ended";
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

    await this.client.emails.send({
      to: req.email,
      from: "andreas@unkey.dev",
      reply_to: this.replyTo,
      subject: "Your Unkey trial has ended",
      html,
    });
  }

  public async sendSubscriptionEnded(req: {
    email: string;
    name: string;
  }): Promise<void> {
    const html = render(<SubscriptionEnded username={req.name} />);

    await this.client.emails.send({
      to: req.email,
      from: "andreas@unkey.dev",
      reply_to: this.replyTo,
      subject: "Your Unkey trial has ended",
      html,
    });
  }

  public async sendWelcomeEmail(req: {
    email: string;
  }): Promise<void> {
    const html = render(<WelcomeEmail />);

    await this.client.emails.send({
      to: req.email,
      from: "james@unkey.dev",
      reply_to: this.replyTo,
      subject: "Welcome to Unkey",
      html,
    });
  }

  public async sendPaymentIssue(req: { email: string; name: string; date: Date }): Promise<void> {
    const html = render(<PaymentIssue username={req.name} date={req.date.toDateString()} />);

    await this.client.emails.send({
      to: req.email,
      from: "andreas@unkey.dev",
      reply_to: this.replyTo,
      subject: "There was an issue with your payment",
      html,
    });
  }
}
