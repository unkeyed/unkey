"use client";
import { CTA } from "@/components/cta";
import { Particles } from "@/components/particles";
import { ShinyCardGroup } from "@/components/shiny-card";
import { TopLeftShiningLight, TopRightShiningLight } from "@/components/svg/hero";
import { Check, Stars } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { useState } from "react";
import {
  BelowEnterpriseSvg,
  Bullet,
  Bullets,
  Button,
  Color,
  Cost,
  EnterpriseCardHighlight,
  FreeCardHighlight,
  PricingCard,
  PricingCardContent,
  PricingCardFooter,
  PricingCardHeader,
  ProCardHighlight,
  Separator,
} from "./components";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "./select";

const buckets: Array<{ price: string; requests: number }> = [
  {
    price: "$25",
    requests: 250_000,
  },
  {
    price: "$50",
    requests: 500_000,
  },
  {
    price: "$75",
    requests: 1_000_000,
  },
  {
    price: "$100",
    requests: 2_000_000,
  },
  {
    price: "$250",
    requests: 10_000_000,
  },
  {
    price: "$500",
    requests: 50_000_000,
  },
  {
    price: "$1000",
    requests: 100_000_000,
  },
];

export default function PricingPage() {
  const { format } = Intl.NumberFormat(undefined, { notation: "compact" });

  const [selectedBucketIndex, setSelectedBucketIndex] = useState(0);
  return (
    <div className="px-4 mx-auto lg:px-0 pt-[64px]">
      <TopRightShiningLight />
      <TopLeftShiningLight />
      <div
        aria-hidden
        className="absolute -top-[4.5rem] left-1/2 -translate-x-1/2 w-[2679px] h-[540px] -scale-x-100"
      >
        <div className="absolute -left-[100px] w-[1400px] aspect-[1400/541] [mask-image:radial-gradient(50%_76%_at_92%_28%,_#FFF_0%,_#FFF_30.03%,_rgba(255,_255,_255,_0.00)_100%)]">
          <Image
            alt="Visual decoration auth chip"
            src="/images/landing/leveled-up-api-auth-chip-min.svg"
            fill
          />
        </div>
        <div className="absolute right-0 w-[1400px] aspect-[1400/541] [mask-image:radial-gradient(26%_76%_at_30%_6%,_#FFF_0%,_#FFF_30.03%,_rgba(255,_255,_255,_0.00)_100%)]">
          <Image
            alt="Visual decoration auth chip"
            src="/images/landing/leveled-up-api-auth-chip-min.svg"
            fill
          />
        </div>
      </div>

      <div className="flex flex-col items-center justify-center my-16 xl:my-24">
        <h1 className="section-title-heading-gradient max-sm:mx-6 max-sm:text-4xl font-medium text-[4rem] leading-[4rem] max-w-xl text-center ">
          Start for free, scale as you go
        </h1>

        {/* <p className="mt-8 bg-gradient-to-br text-transparent bg-gradient-stop bg-clip-text from-white via-white via-40% to-white/30 max-w-lg text-center">
          We wanted pricing to be simple and affordable for anyone, so we've created flexible plans
          that don't need an accounting degree to figure out.
        </p>  */}
      </div>

      <ShinyCardGroup className="grid h-full max-w-4xl grid-cols-2 gap-6 mx-auto group">
        <PricingCard color={Color.White} className="col-span-2 md:col-span-1">
          <FreeCardHighlight className="absolute top-0 right-0 pointer-events-none" />

          <PricingCardHeader
            title="Free Tier"
            description="Everything you need to start!"
            className="bg-gradient-to-tr from-transparent to-[#ffffff]/10 "
            color={Color.White}
          />
          <Separator />

          <PricingCardContent>
            <Cost dollar="$0" />
            <Button label="Start for Free" />
            <Bullets>
              <li>
                <Bullet Icon={Check} label="1k API keys" color={Color.White} />
              </li>

              <li>
                <Bullet Icon={Check} label="150k valid requests / month" color={Color.White} />
              </li>
              <li>
                {" "}
                <Bullet Icon={Check} label="7-day logs retention" color={Color.White} />
              </li>
              <li>
                <Bullet Icon={Check} label="30-day audit log retention" color={Color.White} />
              </li>
              <li>
                <Bullet Icon={Check} label="Unlimited APIs" color={Color.White} />
              </li>
              <li>
                <div className="h-6" />
              </li>
            </Bullets>
          </PricingCardContent>
          <PricingCardFooter>
            <div className="flex flex-col gap-2">
              <p className="text-sm font-bold text-white">What counts as valid? </p>
              <p className="text-xs text-white/60">
                A valid request is a key verification or a ratelimit operation that result in
                proividng access to your service. Requests may be invalid due to exceeding limits,
                keys being expired or disabled, or other factors. To protect your business from
                abuse, we do not charge for invalid requests.
              </p>
              <p className="text-xs text-white/60">
                Only key verification and ratelimiting requests are billable. All regular API
                requests are always free.
              </p>
            </div>
          </PricingCardFooter>
        </PricingCard>
        <PricingCard color={Color.Yellow} className="col-span-2 md:col-span-1">
          <ProCardHighlight className="absolute top-0 right-0 pointer-events-none" />

          <PricingCardHeader
            title="Pro Tier"
            description="Predicatable pricing without surprises."
            className="bg-gradient-to-tr from-black/50 to-[#FFD600]/10 "
            color={Color.Yellow}
          />
          <Separator />

          <PricingCardContent>
            <div className="flex items-center justify-between">
              <Cost dollar={buckets[selectedBucketIndex].price} className="w-full" />

              <Select
                value={selectedBucketIndex.toString()}
                onValueChange={(v) => setSelectedBucketIndex(Number.parseInt(v))}
              >
                <SelectTrigger className="max-w-40">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent className="w-full">
                  {buckets.map((b, i) => (
                    <SelectItem key={b.price} value={i.toString()}>
                      {format(b.requests)} Requests
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <Button label="Get Started with Pro" />
            <Bullets>
              <li>
                <Bullet Icon={Check} label="1M API keys" color={Color.Yellow} />
              </li>
              <li>
                <Bullet
                  Icon={Check}
                  label={`${format(buckets[selectedBucketIndex].requests)} valid requests / month`}
                  color={Color.Yellow}
                />
              </li>
              <li>
                <Bullet Icon={Check} label="30-day logs retention" color={Color.Yellow} />
              </li>
              <li>
                <Bullet Icon={Check} label="90-day audit log retention" color={Color.Yellow} />
              </li>
              <li>
                <Bullet Icon={Check} label="Unlimited APIs" color={Color.Yellow} />
              </li>
              <li>
                <Bullet Icon={Check} label="Workspaces with team members" color={Color.Yellow} />
              </li>
            </Bullets>
          </PricingCardContent>
          <PricingCardFooter>
            <div className="flex flex-col gap-2">
              <p className="text-sm font-bold text-white">What happens when I go over my plan? </p>
              <p className="text-xs text-white/60">
                We want you to succeed, and not be afraid of surprise charges. If you unexpectedly
                go over the limits of your plan, we won't shut you down automatically, nor will we
                charge you extra. If your usage is consistently exceeding the limits, you should
                upgrade to the next higher plan.
              </p>
            </div>
          </PricingCardFooter>
        </PricingCard>

        <PricingCard color={Color.Purple} className="col-span-2">
          <EnterpriseCardHighlight className="absolute top-0 right-0 pointer-events-none" />

          <div className="flex flex-col h-full md:flex-row">
            <div className="flex flex-col w-full gap-8">
              <PricingCardHeader
                title="Enterprise Tier"
                description="Need more support or pricing doesn't work for your business?"
                color={Color.Purple}
                className="bg-gradient-to-tr from-transparent to-[#9D72FF]/10 "
              />
              <PricingCardContent>
                <Cost dollar="Custom $" />
                <Link href="mailto:support@unkey.dev?subject=Unkey Enterprise Quote">
                  <div className="w-full p-px rounded-lg h-10 bg-gradient-to-r from-[#02DEFC] via-[#0239FC] to-[#7002FC] overflow-hidden">
                    <div className="bg-black rounded-[7px] h-full bg-opacity-95 hover:bg-opacity-25 duration-1000">
                      <div className="flex items-center justify-center w-full h-full bg-gradient-to-tr from-[#02DEFC]/20 via-[#0239FC]/20 to-[#7002FC]/20  rounded-[7px]">
                        <span className="text-sm font-semibold text-white">Contact Us</span>
                      </div>
                    </div>
                  </div>
                </Link>
              </PricingCardContent>
            </div>
            <Separator orientation="vertical" className="hidden md:flex" />
            <Separator orientation="horizontal" className="md:hidden" />
            <div className="relative w-full p-8">
              <Particles
                className="absolute inset-0 duration-500 opacity-50 -z-10 group-hover:opacity-100"
                quantity={50}
                color={Color.Purple}
                vx={0.1}
                vy={-0.1}
              />
              <Bullets>
                <li>
                  <Bullet Icon={Check} label="Custom Quotas" color={Color.Purple} />
                </li>
                <li>
                  <Bullet Icon={Check} label="IP Whitelisting" color={Color.Purple} />
                </li>
                <li>
                  <Bullet Icon={Check} label="Dedicated Support" color={Color.Purple} />
                </li>
                <li>
                  <Bullet
                    Icon={Stars}
                    label="More coming soon"
                    color={Color.Purple}
                    textColor="text-white/50"
                  />
                </li>
              </Bullets>
            </div>
          </div>
        </PricingCard>
      </ShinyCardGroup>
      <BelowEnterpriseSvg className="container inset-x-0 top-0 mx-auto -mt-64 -mb-32" />

      <div className="-mx-4 lg:mx-0">
        <CTA />
      </div>
    </div>
  );
}
