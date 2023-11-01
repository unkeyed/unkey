"use client";
import { Resend as Client } from "resend";

import { render } from "@react-email/render";
import React from "react";
import { PaymentIssue } from "../emails/payment_issue";
import { SubscriptionEnded } from "../emails/subscription_ended";
import { TrialEndsIn3Days } from "../emails/trial_ends_in_3_days";

export class Resend {
  private readonly apiKey: string;
  private readonly baseUrl: string;
  private readonly client: Client;
  private readonly from = "updates.unkey.dev";
  private readonly replyTo = "support@unkey.dev";

  constructor(opts: { apiKey: string }) {
    this.client = new Client(opts.apiKey);
    // TODO: remove this after resend added audiences to the sdk
    this.apiKey = opts.apiKey;
    this.baseUrl = "https://api.resend.com";
  }
  // TODO: remove
  private async fetch<TResult>(req: {
    path: string[];
    method: "GET" | "POST" | "PUT" | "DELETE";
    body?: unknown;
  }): Promise<TResult> {
    const url = `${this.baseUrl}/${req.path.join("/")}`;

    const res = await fetch(url, {
      method: req.method,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.apiKey}`,
      },
      body: JSON.stringify(req.body),
    });
    if (res.ok) {
      return await res.json();
    }
    throw new Error(`error from api: ${await res.text()}`);
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
      from: this.from,
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
      from: this.from,
      reply_to: this.replyTo,
      subject: "Your Unkey trial has ended",
      html,
    });
  }

  public async sendPaymentIssue(req: { email: string; name: string; date: Date }): Promise<void> {
    const html = render(<PaymentIssue username={req.name} date={req.date.toDateString()} />);

    await this.client.emails.send({
      to: req.email,
      from: this.from,
      reply_to: this.replyTo,
      subject: "There was an issue with your payment",
      html,
    });
  }

  public async addUserToAudience(req: {
    email: string;
    audienceId: string;
    firstName?: string;
    lastName?: string;
  }): Promise<void> {
    await this.fetch({
      path: ["audiences", req.audienceId, "contacts"],
      method: "POST",
      body: {
        email: req.email,
        first_name: req.firstName,
        last_name: req.lastName,
      },
    });
  }
}
