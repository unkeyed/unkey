import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";
import { Feature, FeatureContent, FeatureHeader, FeatureIcon, FeatureTitle } from "./feature";

import * as React from "react";

const FeatureGrid = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className }, ref) => (
    <div ref={ref} className={cn(className)}>
      <div className="grid grid-rows gap-4 rounded-3xl border-[0.75px] border-[rgba(255,255,255,0.15)] p-10 lg:w-full">
        <div className="grid md:grid-cols-3 p-[16px,0px,0px,0px]">
          <Feature>
            <FeatureHeader>
              <FeatureIcon iconName="cloud" />
              <FeatureTitle>Multi-Cloud</FeatureTitle>
            </FeatureHeader>
            <FeatureContent>
              Seamlessly scale and distribute your services with multi-cloud support, allowing you
              to use unkey with any cloud provider effortlessly.
            </FeatureContent>
          </Feature>
          <Feature>
            <FeatureHeader>
              <FeatureIcon iconName="data" />
              <FeatureTitle>Data Retention</FeatureTitle>
            </FeatureHeader>
            <FeatureContent>
              Align data storage with compliance goals, customize retention for regulatory
              requirements, and effortlessly maintain data integrity.
            </FeatureContent>
          </Feature>
          <Feature>
            <FeatureHeader>
              <FeatureIcon iconName="api" />
              <FeatureTitle>API-first / UI-first</FeatureTitle>
            </FeatureHeader>
            <FeatureContent>
              Enjoy flexibility and control with both API and user-friendly UI access, ensuring a
              smooth experience for developers and non-technical users alike.
            </FeatureContent>
          </Feature>
        </div>
        <div className="p-5 max-sm:hidden">
          {" "}
          <Separator orientation="horizontal" className="ml-8 m-0" />
        </div>
        <div className="grid md:grid-cols-3 space-x-2">
          <Feature>
            <FeatureHeader>
              <FeatureIcon iconName="role" />
              <FeatureTitle>Role-Based Access Control</FeatureTitle>
            </FeatureHeader>
            <FeatureContent>
              Fine-tune access privileges with role-based control per key, enabling precise
              management and ensuring security at every level.
            </FeatureContent>
          </Feature>
          <Feature>
            <FeatureHeader>
              <FeatureIcon iconName="detect" />
              <FeatureTitle>Detect and Protect</FeatureTitle>
            </FeatureHeader>
            <FeatureContent>
              Take immediate control over your system's security with the ability to instantly
              revoke access keys, providing swift response to potential threats.
            </FeatureContent>
          </Feature>
          <Feature>
            <FeatureHeader>
              <FeatureIcon iconName="sdk" />
              <FeatureTitle>SDKs</FeatureTitle>
            </FeatureHeader>
            <FeatureContent>
              Accelerate development with Software Development Kits (SDKs), providing tools for
              seamless integration processes.
            </FeatureContent>
          </Feature>
        </div>
        <div className="p-5 max-sm:hidden">
          <Separator orientation="horizontal" className="p-0 m-0" />
        </div>
        <div className="grid md:grid-cols-3">
          <Feature>
            <FeatureHeader>
              <FeatureIcon iconName="vercel" />
              <FeatureTitle>Vercel Integration</FeatureTitle>
            </FeatureHeader>
            <FeatureContent>
              Effortlessly deploy applications with Vercel integration, streamlining the
              development-to-deployment pipeline for optimal efficiency.
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
              Collaborate effectively within your organization by organizing teams, streamlining
              communication for group-based projects.
            </FeatureContent>
          </Feature>
        </div>
      </div>
    </div>
  ),
);
FeatureGrid.displayName = "FeatureGrid";

export { FeatureGrid };
