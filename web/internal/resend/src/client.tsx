import { Resend as Client } from "resend";

import { render } from "@react-email/render";
// biome-ignore lint/correctness/noUnusedImports: React UMD bypass
import React from "react";
import { ApiV1Migration } from "../emails/api_v1_migration";
import { PaymentIssue } from "../emails/payment_issue";
import { SecretScanningKeyDetected } from "../emails/secret_scanning_key_detected";
import { WelcomeEmail } from "../emails/welcome_email";
export class Resend {
  public readonly client: Client;
  private readonly replyTo = "support@unkey.com";

  constructor(opts: { apiKey: string }) {
    this.client = new Client(opts.apiKey);
  }

  public async sendWelcomeEmail(req: { email: string }) {
    const fiveMinutesFromNow = new Date(Date.now() + 5 * 60 * 1000).toISOString();

    const html = await render(<WelcomeEmail />);
    try {
      const result = await this.client.emails.send({
        to: req.email,
        from: "James from Unkey <james@updates.unkey.com>",
        replyTo: this.replyTo,
        subject: "Welcome to Unkey",
        html,
        scheduledAt: fiveMinutesFromNow,
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
    const html = await render(<PaymentIssue username={req.name} date={req.date.toDateString()} />);
    try {
      const result = await this.client.emails.send({
        to: req.email,
        from: "James from Unkey <james@updates.unkey.com>",
        replyTo: this.replyTo,
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
  public async sendLeakedKeyEmail(req: {
    email: string;
    date: string;
    source: string;
    url: string;
  }): Promise<void> {
    const { date, email, source, url } = req;
    const html = await render(<SecretScanningKeyDetected date={date} source={source} url={url} />);

    try {
      const result = await this.client.emails.send({
        to: email,
        from: "James from Unkey <james@updates.unkey.com>",
        replyTo: this.replyTo,
        subject: "Unkey root key exposed in public Github repository",
        html: html,
      });
      if (!result.error) {
        return;
      }
      throw result.error;
    } catch (error) {
      console.error(error);
    }
  }

  public async sendApiV1MigrationEmail(req: {
    email: string;
    name: string;
    workspace: string;
    deprecatedEndpoints: string[];
  }): Promise<void> {
    const html = await render(
      <ApiV1Migration
        username={req.name}
        workspaceName={req.workspace}
        deprecatedEndpoints={req.deprecatedEndpoints}
      />,
    );
    try {
      const result = await this.client.emails.send({
        to: req.email,
        from: "Andreas from Unkey <andreas@updates.unkey.com>",
        replyTo: this.replyTo,
        subject: "Action Required: Migrate from API v1 to v2 - Deadline January 1st, 2026",
        html,
      });

      if (!result.error) {
        return;
      }
      throw result.error;
    } catch (error) {
      console.error("Error occurred sending API v1 migration email ", JSON.stringify(error));
    }
  }
}
