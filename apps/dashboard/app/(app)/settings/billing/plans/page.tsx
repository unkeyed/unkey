import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Gear } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { ArrowLeft } from "lucide-react";
import Link from "next/link";
import { notFound } from "next/navigation";
import { ChangePlanButton } from "./button";
import { tiers } from "./constants";

export default async function Page() {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.tenantId, tenantId),
  });
  if (!workspace) {
    return notFound();
  }

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gear />}>
          <Navbar.Breadcrumbs.Link href="/settings/billing">Billing</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href="/settings/billing/plans" active>
            Plans
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
      </Navbar>
      <PageContent>
        <div>
          <Link
            href="/settings/billing"
            className="flex items-center gap-1 text-sm duration-200 text-content-subtle hover:text-foreground"
          >
            <ArrowLeft className="w-4 h-4" /> Back to Billing
          </Link>

          <div className="flex flex-col gap-6 mt-6 md:flex-col lg:flex-row">
            {(["free", "pro", "custom"] as const).map((tier) => (
              <div
                key={tier}
                className={
                  "border border-border flex w-full flex-col justify-between rounded-lg bg-white dark:bg-black p-8 lg:w-1/3 xl:p-10"
                }
              >
                <div className="flex items-center justify-between gap-x-4">
                  <h2 id={tier} className={"text-content text-2xl font-semibold leading-8"}>
                    {tiers[tier].name}
                  </h2>
                </div>
                <p className="mt-4 min-h-[3rem] text-sm leading-6 text-content-subtle">
                  {tiers[tier].description}
                </p>
                <p className="flex items-center mx-auto mt-6 gap-x-1">
                  {typeof tiers[tier].price === "number" ? (
                    <>
                      <span className="text-4xl font-bold tracking-tight text-center text-content">
                        {`$${tiers[tier].price}`}
                      </span>
                      <span className="mx-auto text-sm font-semibold leading-6 text-center text-content-subtle">
                        {"/month"}
                      </span>
                    </>
                  ) : (
                    <span className="mx-auto text-4xl font-bold tracking-tight text-center text-content">
                      {tiers[tier].price}
                    </span>
                  )}
                </p>

                <div className="flex flex-col justify-between grow">
                  <ul className="mt-8 space-y-3 text-sm leading-6 text-content-subtle xl:mt-10">
                    {tiers[tier].features.map((feature) => (
                      <li key={feature} className="flex gap-x-3">
                        <svg
                          xmlns="http://www.w3.org/2000/svg"
                          viewBox="0 0 24 24"
                          className="flex-none w-5 h-6 text-gray-700"
                          aria-hidden="true"
                        >
                          <path
                            fill="currentColor"
                            fillRule="evenodd"
                            d="M19.916 4.626a.75.75 0 0 1 .208 1.04l-9 13.5a.75.75 0 0 1-1.154.114l-6-6a.75.75 0 0 1 1.06-1.06l5.353 5.353l8.493-12.739a.75.75 0 0 1 1.04-.208Z"
                            clipRule="evenodd"
                          />
                        </svg>

                        {feature}
                      </li>
                    ))}
                  </ul>
                  {tiers[tier].footnotes && (
                    <ul className="mt-6 mb-8">
                      {tiers[tier].footnotes.map((footnote, i) => (
                        <li
                          // biome-ignore lint/suspicious/noArrayIndexKey: I got nothing better right now
                          key={`note-${i}`}
                          className="flex text-xs text-content-subtle gap-x-3"
                        >
                          {footnote}
                        </li>
                      ))}
                    </ul>
                  )}
                  {tier === "custom" ? (
                    <Link href="mailto:support@unkey.dev">
                      <Button
                        className="w-full col-span-1"
                        variant="primary"
                        disabled={workspace.plan === "enterprise"}
                      >
                        Schedule a Call
                      </Button>
                    </Link>
                  ) : (
                    <ChangePlanButton
                      workspace={workspace}
                      newPlan={tier}
                      label={tiers[tier].buttonText}
                    />
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      </PageContent>
    </div>
  );
}
