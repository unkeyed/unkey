import { cn } from "@/lib/utils";
import { Feature, FeatureContent, FeatureHeader, FeatureIcon, FeatureTitle } from "./feature";

import * as React from "react";

const FeatureGrid = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className }, ref) => (
    <div ref={ref} className={cn(className)}>
      <div className="grid grid-rows gap-4 rounded-3xl border-[0.75px] border-[rgba(255,255,255,0.15)] p-10 lg:w-full">
        <div className="feature-grid md:grid-cols-2 xl:grid-cols-3 p-[16px,0px,0px,0px]">
          <Feature>
            <FeatureHeader>
              <FeatureIcon iconName="cloud" />
              <FeatureTitle>Multi-Cloud</FeatureTitle>
            </FeatureHeader>
            <FeatureContent>
              Unkey works with any cloud provider, ensuring a fast global experience regardless of
              your choice of infrastructure.
            </FeatureContent>
          </Feature>
          <Feature>
            <FeatureHeader>
              <FeatureIcon iconName="data" />
              <FeatureTitle>Rate limiting</FeatureTitle>
            </FeatureHeader>
            <FeatureContent>
              Protect your APIs with simple, configurable rate limiting. Unkey’s global rate
              limiting requires zero setup and allows for custom configuration per customer.
            </FeatureContent>
          </Feature>
          <Feature>
            <FeatureHeader>
              <FeatureIcon iconName="api" />
              <FeatureTitle>API-first / UI-first</FeatureTitle>
            </FeatureHeader>
            <FeatureContent>
              Unkey is designed to be equally usable via its API and dashboard, ensuring a smooth
              experience for developers and non-technical users alike.
            </FeatureContent>
          </Feature>
          <Feature>
            <FeatureHeader>
              <FeatureIcon iconName="role" />
              <FeatureTitle>Role-based access control</FeatureTitle>
            </FeatureHeader>
            <FeatureContent>
              Granular access privileges with either role or permission-based control. Permission
              changes are propagated globally in seconds.
            </FeatureContent>
          </Feature>
          <Feature>
            <FeatureHeader>
              <FeatureIcon iconName="detect" />
              <FeatureTitle>Proactive protection</FeatureTitle>
            </FeatureHeader>
            <FeatureContent>
              Take immediate control over your system's security with the ability to instantly
              revoke access , providing swift response to potential threats.
            </FeatureContent>
          </Feature>
          <Feature>
            <FeatureHeader>
              <FeatureIcon iconName="sdk" />
              <FeatureTitle>SDKs</FeatureTitle>
            </FeatureHeader>
            <FeatureContent>
              Hit the ground running and accelerate development with SDKs in the language of your
              choice.
            </FeatureContent>
          </Feature>
          {/* <div className="p-5 max-sm:hidden">
          <Separator orientation="horizontal" className="p-0 m-0" />
        </div> */}
          <Feature>
            <FeatureHeader>
              <FeatureIcon iconName="vercel" />
              <FeatureTitle>Vercel Integration</FeatureTitle>
            </FeatureHeader>
            <FeatureContent>
              Deploy applications with our official Vercel integration, streamlining the
              development-to-deployment pipeline.
            </FeatureContent>
          </Feature>
          <Feature>
            <FeatureHeader>
              <FeatureIcon iconName="automatic" />
              <FeatureTitle>Automatic Key Expiration</FeatureTitle>
            </FeatureHeader>
            <FeatureContent>
              Simplify key management with automatic key expiration, reducing the risk of
              unauthorized access over time.
            </FeatureContent>
          </Feature>
          <Feature>
            <FeatureHeader>
              <FeatureIcon iconName="usage" />
              <FeatureTitle>Usage Limits per Key</FeatureTitle>
            </FeatureHeader>
            <FeatureContent>
              Create keys with a fixed amount of usage and the ability to refill periodically,
              limiting the potential for abuse and allowing for usage-based billing with credits.
            </FeatureContent>
          </Feature>
        </div>
      </div>
    </div>
  ),
);
FeatureGrid.displayName = "FeatureGrid";

export { FeatureGrid };
