import EarlyAccessInvitation from "./emails/EarlyAccessInvitation";
import { render } from "@react-email/render";
import { Resend } from "resend";
import { ResetWarning } from "./emails/ResetWarning";

export class Email {
  private client: Resend;

  constructor(opts: { apiKey: string }) {
    this.client = new Resend(opts.apiKey);
  }

  public async sendEarlyAccessInvitation(opts: {
    to: string;
    inviteLink: string;
  }) {
    const html = render(<EarlyAccessInvitation inviteLink={opts.inviteLink} />);
    return await this.client.sendEmail({
      from: "hello@unkey.dev",
      to: opts.to,
      reply_to: "unkey@chronark.com",
      subject: "You're invited to join the unkey.dev early access",
      html,
    });
  }

  public async sendResetWarning(opts: {
    to: string;
    name: string;
  }) {
    const html = render(<ResetWarning name={opts.name} />);
    return await this.client.sendEmail({
      from: "andreas@unkey.dev",
      to: opts.to,
      reply_to: "unkey@chronark.com",
      subject: "Unkey: API Key Reset (Sunday)",
      html,
    });
  }
}
