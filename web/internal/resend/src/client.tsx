import { Resend as Client } from "resend";

import { render, toPlainText } from "@react-email/render";
import { PaymentIssue } from "../emails/payment_issue";
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
        text: toPlainText(html),
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
    workspaceSlug: string;
  }): Promise<void> {
    const html = await render(
      <PaymentIssue
        username={req.name}
        date={req.date.toDateString()}
        workspaceSlug={req.workspaceSlug}
      />,
    );
    try {
      const result = await this.client.emails.send({
        to: req.email,
        from: "James from Unkey <james@updates.unkey.com>",
        replyTo: this.replyTo,
        subject: "There was an issue with your payment",
        html,
        text: toPlainText(html),
      });
      if (!result.error) {
        return;
      }
      throw result.error;
    } catch (error) {
      console.error("Error occurred sending payment issue email ", JSON.stringify(error));
    }
  }
}
