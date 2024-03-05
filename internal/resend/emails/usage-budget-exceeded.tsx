"use client";
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

export type Props = {
  workspace: string;
  budgetedAmount: number;
  currentPeriodBilling: number;
};

export function UsageBudgetExceeded({
  workspace = "Ubinatus",
  budgetedAmount = 100,
  currentPeriodBilling = 120,
}: Props) {
  return (
    <Tailwind>
      <Head />
      <Html className="font-sans text-zinc-800">
        <Section className="bg-white">
          <Container className="container mx-auto">
            <Heading className="font-sans text-2xl text-semibold">Budget Notification</Heading>
            <Text>Hey,</Text>
            <Text>
              Hope you're doing awesome! Just a quick heads up from your friends at Unkey — looks
              like your spending has just nudged past your set budget. Oops! 🙈
            </Text>

            <Text>
              <b>Workspace: </b>${workspace}
            </Text>
            <Text>
              <b>Your Budget: </b>${budgetedAmount}
            </Text>
            <Text style={{ marginTop: -5 }}>
              <b>Current Spend: </b>${currentPeriodBilling}
            </Text>
            <Text style={{ marginTop: -5 }}>
              <b>Over by: </b>${currentPeriodBilling - budgetedAmount}
            </Text>

            <Text>
              {"You can find firther details by accessing your "}
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

export default UsageBudgetExceeded;
