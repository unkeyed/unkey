import { Container } from "@react-email/container";
import { Head } from "@react-email/head";
import { Heading } from "@react-email/heading";
import { Hr } from "@react-email/hr";
import { Html } from "@react-email/html";
import { Link } from "@react-email/link";
import { Section } from "@react-email/section";
import { Tailwind } from "@react-email/tailwind";
import { Text } from "@react-email/text";
import React from "react";
import tailwindConfig from "../tailwind.config";

export type Props = {
  workspace: string;
  budgetedAmount: number;
  currentPeriodBilling: number;
};

function UsageBudgetExceeded({ workspace, budgetedAmount, currentPeriodBilling }: Props) {
  return (
    <Tailwind config={tailwindConfig}>
      <Html className="font-sans text-zinc-800">
        <Head />
        <Section className="bg-white">
          <Container className="container mx-auto">
            <Heading className="font-sans text-2xl text-semibold">Budget Notification</Heading>
            <Text>Hey,</Text>
            <Text>
              Hope you're doing awesome! Just a quick heads up from your friends at Unkey — looks
              like your spending has just nudged past your set budget.
            </Text>

            <Text className="mb-1">
              <b>Workspace: </b>
              {workspace}
            </Text>
            <Text className="my-1">
              <b>Your Budget: </b>${budgetedAmount}
            </Text>
            <Text className="my-1">
              <b>Current Spend: </b>${currentPeriodBilling}
            </Text>
            <Text className="mt-1">
              <b>Over by: </b>${currentPeriodBilling - budgetedAmount}
            </Text>

            <Text>
              {"You can find further details by accessing your "}
              <Link href="https://unkey.dev/app/settings/billing">Unkey Billing dashboard</Link>.
            </Text>

            <Text>
              Cheers,
              <br />
              Andreas
            </Text>

            <Hr />
            <Text>
              {"P.S. Love seeing how you’re using Unkey. Let’s keep the good vibes rolling!"}
            </Text>
          </Container>
        </Section>
      </Html>
    </Tailwind>
  );
}

UsageBudgetExceeded.PreviewProps = {
  workspace: "Mugiwara Crew",
  budgetedAmount: 100,
  currentPeriodBilling: 115,
} satisfies Props;

export default UsageBudgetExceeded;
