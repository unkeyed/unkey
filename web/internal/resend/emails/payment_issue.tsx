import { Fragment } from "react";
import { Emphasis } from "../src/components/emphasis";
import { Layout } from "../src/components/layout";

export type Props = {
  username: string;
  date: string;
  workspaceSlug: string;
};

export function PaymentIssue({ username, date, workspaceSlug }: Props) {
  const workspaceURL = `https://app.unkey.com/${workspaceSlug}/settings/billing`;
  return (
    <Layout
      title="Update your payment information"
      preview={`We couldn't process your payment on ${date}. Update your payment information to prevent a downgrade.`}
      body={[
        <Fragment key="payment">
          {username}, we couldn't process your payment on <Emphasis>{date}</Emphasis>. Update your
          payment information to prevent your account from being downgraded.
        </Fragment>,
      ]}
      action={{ label: "Update payment information", href: workspaceURL }}
    />
  );
}

PaymentIssue.PreviewProps = {
  username: "Mr. Pilkington",
  date: "Tue Oct 01 2024",
  workspaceSlug: "demo-workspace",
} satisfies Props;

// biome-ignore lint/style/noDefaultExport: Too scared to modify that one
export default PaymentIssue;
