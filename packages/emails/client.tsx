import EarlyAccessInvitation from "./emails/EarlyAccessInvitation";
import { render } from "@react-email/render";
import { Resend } from "resend";

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
}
