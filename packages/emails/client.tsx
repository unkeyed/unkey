import EarlyAccessInvitation from "./emails/EarlyAccessInvitation";
import { render } from "@react-email/render";
import { Resend } from "resend";
import { ResetWarning } from "./emails/ResetWarning";
import Changelog from "./emails/Changelog";

interface ChangelogProps {
  item?: { id: number; shortDescription: string }[];
}
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

  public async sendChangelog(opts: {
    to: string;
    preview: string;
    changelog: ChangelogProps;
    date: string;
    changelogUrl: string;
  }) {
    const html = render(<Changelog preview={opts.preview} changelog={opts.changelog} date={opts.date} changelogUrl={opts.changelogUrl}  />);
    return await this.client.sendEmail({
      from: "james@unkey.dev",
      to: opts.to,
      reply_to: "james@unkey.dev",
      subject: opts.preview,
      html,
    });
  }
}