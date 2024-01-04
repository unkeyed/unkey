import Link from "next/link";

const tiers = {
  free: {
    name: "Free Tier",
    id: "free",
    href: "/app",
    price: 0,
    description: "Everything you need to start your next API!",
    features: [
      "100 Monthly Active Keys",
      "2500 Successful Verifications per month",
      "Unlimited APIs",
      "7 Days Analytics Retention",
    ],
    footnotes: [],
  },
  pro: {
    name: "Pro Tier",
    id: "paid",
    href: "/app",
    price: 25,
    description: "For those with teams and more demanding needs",
    buttonText: "Upgrade now",
    features: [
      "250 Monthly Active keys included *",
      "150,000 Successful Verifications included **",
      "Unlimited APIs",
      "Workspaces with team members",
      "90 Days Analytics Retention",
      "90 Days Audit Log Retention",
    ],
    footnotes: [
      " *  Additional active keys are billed at $0.10",
      " ** Additional verifications are billed at $10 per 100,000",
    ],
  },
  custom: {
    name: "Custom",
    id: "enterprise",
    href: "https://cal.com/team/unkey/unkey-chat",
    price: "Let's talk",
    description: "We offer custom pricing for those with volume needs",
    buttonText: "Schedule a call",
    features: [
      "Custom Verification Limits",
      "Custom Active Key Limits",
      "Pricing based on your needs",
      "Custom Analytics Retention",
      "Dedicated support contract",
      "Whitelist IP per API",
    ],
    footnotes: [],
  },
};

export const metadata = {
  title: "Pricing | Unkey",
  description: "Pricing for Unkey",
  openGraph: {
    title: "Pricing | Unkey",
    description: "Pricing for Unkey",
    url: "https://unkey.dev/pricing",
    siteName: "unkey.dev",
    images: [
      {
        url: "https://unkey.dev/images/landing/og.png",
        width: 1200,
        height: 675,
      },
    ],
  },
  twitter: {
    title: "Unkey",
    card: "summary_large_image",
  },
  icons: {
    shortcut: "/images/landing/unkey.png",
  },
};

export default async function PricingPage() {
  return (
    <div>
      <div className="flex items-center justify-center overflow-auto">
        <div className="relative w-full max-w-6xl px-6 py-10 isolate lg:px-8">
          <div>
            <div className="text-center">
              <h1 className="text-4xl font-bold tracking-tight text-gray-900 sm:text-6xl">
                {"Pricing built for "}
                <br />
                <span className="text-gray-500">{"everyone."}</span>
              </h1>
              <p className="mt-6 text-lg leading-8 text-gray-600">
                {
                  "We wanted pricing to be simple and affordable for anyone, so we've created a flexible plans that don't need an accounting degree to figure out."
                }
              </p>
            </div>
            <div className="flex flex-col mt-10 gap-y-6 sm:gap-x-6 lg:flex-row">
              {(["free", "pro", "custom"] as const).map((tier) => (
                <div
                  key={tiers[tier].id}
                  className={
                    "ring-1 ring-gray-200 flex w-full flex-col justify-between rounded-3xl bg-white p-8 shadow-lg lg:w-1/3 xl:p-10"
                  }
                >
                  <div className="flex items-center justify-between gap-x-4">
                    <h2
                      id={tiers[tier].id}
                      className={"text-gray-900 text-2xl font-semibold leading-8"}
                    >
                      {tiers[tier].name}
                    </h2>
                  </div>
                  <p className="mt-4 min-h-[3rem] text-sm leading-6 text-gray-600 tex">
                    {tiers[tier].description}
                  </p>
                  <p className="flex items-center mx-auto mt-6 gap-x-1">
                    {typeof tiers[tier].price === "number" ? (
                      <>
                        <span className="text-4xl font-bold tracking-tight text-center text-gray-900">
                          {`$${tiers[tier].price}`}
                        </span>
                        <span className="text-sm font-semibold leading-6 text-gray-600 mx-autotext-center">
                          {"/month"}
                        </span>
                      </>
                    ) : (
                      <span className="mx-auto text-4xl font-bold tracking-tight text-center text-gray-900">
                        {tiers[tier].price}
                      </span>
                    )}
                  </p>
                  {tier === "custom" ? (
                    <div className="flex justify-center mt-10 group">
                      <Link
                        href={tiers[tier].href}
                        target="_blank"
                        title=""
                        className="relative inline-flex items-center justify-center px-8 py-4 text-lg font-bold text-white transition-all duration-200 bg-gray-900 rounded-lg font-pj focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-gray-900"
                        role="button"
                      >
                        Schedule a Call
                      </Link>
                    </div>
                  ) : (
                    <>
                      <div className="flex justify-center mt-10 group">
                        <Link
                          href={tiers[tier].href}
                          target="_blank"
                          title=""
                          className="relative inline-flex items-center justify-center px-8 py-4 text-lg font-bold text-white transition-all duration-200 bg-gray-900 rounded-lg font-pj focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-gray-900"
                          role="button"
                        >
                          Start Building
                        </Link>
                      </div>
                    </>
                  )}
                  <div className="flex flex-col justify-between grow">
                    <ul className="mt-8 space-y-3 text-sm leading-6 text-gray-600 xl:mt-10">
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
                      <ul className="mt-6">
                        {tiers[tier].footnotes.map((footnote, i) => (
                          <li key={`note-${i}`} className="flex text-xs text-gray-600 gap-x-3">
                            {footnote}
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
