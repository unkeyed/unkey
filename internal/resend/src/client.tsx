import { Resend as Client } from "resend";

import { render } from "@react-email/render";
import React from "react";
import { PaymentIssue } from "../emails/payment_issue";
import { SubscriptionEnded } from "../emails/subscription_ended";
import { TrialEndsIn3Days } from "../emails/trial_ends_in_3_days";
import { WelcomeEmail } from "../emails/welcome_email";
export class Resend {
  public readonly client: Client;
  private readonly domain = "updates.unkey.dev";
  private readonly replyTo = "support@unkey.dev";

  constructor(opts: { apiKey: string }) {
    this.client = new Client(opts.apiKey);
  }

  public async sendTrialEnds(req: {
    email: string;
    name: string;
    workspace: string;
    date: Date;
  }): Promise<void> {
    const html = render(
      <TrialEndsIn3Days
        username={req.name}
        workspaceName={req.workspace}
        endDate={req.date.toDateString()}
      />,
    );

    await this.client.emails.send({
      to: req.email,
      from: `andreas@${this.domain}`,
      reply_to: this.replyTo,
      subject: "Your Unkey trial ends in 3 days",
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
      from: `andreas@${this.domain}`,
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
      from: `james@${this.domain}`,
      reply_to: this.replyTo,
      subject: "Welcome to Unkey",
      html,
    });
  }

  public async sendPaymentIssue(req: { email: string; name: string; date: Date }): Promise<void> {
    const html = render(<PaymentIssue username={req.name} date={req.date.toDateString()} />);

    await this.client.emails.send({
      to: req.email,
      from: `andreas@${this.domain}`,
      reply_to: this.replyTo,
      subject: "There was an issue with your payment",
      html,
    });
  }
}
