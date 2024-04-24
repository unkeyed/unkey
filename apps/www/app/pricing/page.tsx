import { Particles } from "@/components/particles";
import { ShinyCardGroup } from "@/components/shiny-card";
import { Check, Stars } from "lucide-react";
import Link from "next/link";
import { Discover } from "./discover";

import { HeroSvg } from "./hero-svgs";

import { CTA } from "@/components/cta";
import {
  Asterisk,
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

export default function PricingPage() {
  return (
    <div className="px-4 mx-auto lg:px-0 pt-[64px]">
      <HeroSvg className="absolute inset-x-0 top-0 pointer-events-none" />

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
                <Bullet
                  Icon={Check}
                  label="2.5k successful verifications / month"
                  color={Color.White}
                />
              </li>
              <li>
                <Bullet
                  Icon={Check}
                  label="100k successful ratelimits / month"
                  color={Color.White}
                />
              </li>
              <li>
                <Bullet Icon={Check} label="7-day analytics retention" color={Color.White} />
              </li>
              <li>
                <Bullet Icon={Check} label="Unlimited APIs" color={Color.White} />
              </li>
            </Bullets>
          </PricingCardContent>
          <PricingCardFooter>
            <div className="flex flex-col gap-2">
              <p className="text-sm font-bold text-white">What counts as successful? </p>
              <p className="text-xs text-white/60">
                A successful request means everything is fine and you should grant access to the
                user. Requests may be unsuccessful due to exceeding limits, keys being expired or
                disabled, or other factors.
              </p>
              <p className="text-xs text-white/60">
                To protect your business from abuse, we do not charge for unsuccessful requests.
              </p>
            </div>
          </PricingCardFooter>
        </PricingCard>
        <PricingCard color={Color.Yellow} className="col-span-2 md:col-span-1">
          <ProCardHighlight className="absolute top-0 right-0 pointer-events-none" />

          <PricingCardHeader
            title="Pro Tier"
            description="For growing teams with powerful demands"
            className="bg-gradient-to-tr from-black/50 to-[#FFD600]/10 "
            color={Color.Yellow}
          />
          <Separator />

          <PricingCardContent>
            <Cost dollar="$25" />
            <Button label="Get Started with Pro" />
            <Bullets>
              <li>
                <Bullet Icon={Check} label="1M API keys" color={Color.Yellow} />
              </li>
              <li>
                <Bullet
                  Icon={Check}
                  label="150k successful verifications / month"
                  color={Color.Yellow}
                />
              </li>
              <li>
                <Bullet
                  Icon={Check}
                  label="2.5M successful ratelimits / month"
                  color={Color.Yellow}
                />
              </li>
              <li>
                <Bullet Icon={Check} label="90-day analytics retention" color={Color.Yellow} />
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
              <li>
                <Bullet
                  Icon={Stars}
                  label="More coming soon"
                  color={Color.Yellow}
                  textColor="text-white/50"
                />
              </li>
            </Bullets>
          </PricingCardContent>
          <PricingCardFooter>
            <div className="flex flex-col gap-2">
              <Asterisk tag="$1" label="/ additional 10k successful verifications" />
              <Asterisk tag="$1" label="/ additional 100k successful ratelimits" />
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

      <Discover />

      <div className="-mx-4 lg:mx-0">
        <CTA />
      </div>
    </div>
  );
}
