import { Feature, FeatureContent, FeatureHeader, FeatureIcon, FeatureTitle } from "./feature-cell";

import * as React from "react";

const FeatureGrid = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(() => (
  <div className="grid grid-cols-1 md:grid-cols-3 gap-4 rounded-3xl border-[0.75px] border-[rgba(255,255,255,0.15)] pt-10 pr-12 pb-12 pl-12 lg:w-[1192px]">
    <Feature>
      <FeatureHeader>
        <FeatureIcon iconName="cloud" />
        <FeatureTitle>Multi-Cloud</FeatureTitle>
      </FeatureHeader>
      <FeatureContent>
        Seamlessly scale and distribute your services with multi-cloud support, allowing you to use
        unkey with any cloud provider effortlessly.
      </FeatureContent>
    </Feature>
    <Feature>
      <FeatureHeader>
        <FeatureIcon iconName="data" />
        <FeatureTitle>Multi-Cloud</FeatureTitle>
      </FeatureHeader>
      <FeatureContent>
        Seamlessly scale and distribute your services with multi-cloud support, allowing you to use
        unkey with any cloud provider effortlessly.
      </FeatureContent>
    </Feature>
    <Feature>
      <FeatureHeader>
        <FeatureIcon iconName="api" />
        <FeatureTitle>Multi-Cloud</FeatureTitle>
      </FeatureHeader>
      <FeatureContent>
        Seamlessly scale and distribute your services with multi-cloud support, allowing you to use
        unkey with any cloud provider effortlessly.
      </FeatureContent>
    </Feature>
    <Feature>
      <FeatureHeader>
        <FeatureIcon iconName="role" />
        <FeatureTitle>Multi-Cloud</FeatureTitle>
      </FeatureHeader>
      <FeatureContent>
        Seamlessly scale and distribute your services with multi-cloud support, allowing you to use
        unkey with any cloud provider effortlessly.
      </FeatureContent>
    </Feature>
    <Feature>
      <FeatureHeader>
        <FeatureIcon iconName="detect" />
        <FeatureTitle>Multi-Cloud</FeatureTitle>
      </FeatureHeader>
      <FeatureContent>
        Seamlessly scale and distribute your services with multi-cloud support, allowing you to use
        unkey with any cloud provider effortlessly.
      </FeatureContent>
    </Feature>
    <Feature>
      <FeatureHeader>
        <FeatureIcon iconName="sdk" />
        <FeatureTitle>Multi-Cloud</FeatureTitle>
      </FeatureHeader>
      <FeatureContent>
        Seamlessly scale and distribute your services with multi-cloud support, allowing you to use
        unkey with any cloud provider effortlessly.
      </FeatureContent>
    </Feature>
    <Feature>
      <FeatureHeader>
        <FeatureIcon iconName="vercel" />
        <FeatureTitle>Multi-Cloud</FeatureTitle>
      </FeatureHeader>
      <FeatureContent>
        Seamlessly scale and distribute your services with multi-cloud support, allowing you to use
        unkey with any cloud provider effortlessly.
      </FeatureContent>
    </Feature>
    <Feature>
      <FeatureHeader>
        <FeatureIcon iconName="automatic" />
        <FeatureTitle>Multi-Cloud</FeatureTitle>
      </FeatureHeader>
      <FeatureContent>
        Seamlessly scale and distribute your services with multi-cloud support, allowing you to use
        unkey with any cloud provider effortlessly.
      </FeatureContent>
    </Feature>
    <Feature>
      <FeatureHeader>
        <FeatureIcon iconName="usage" />
        <FeatureTitle>Multi-Cloud</FeatureTitle>
      </FeatureHeader>
      <FeatureContent>
        Seamlessly scale and distribute your services with multi-cloud support, allowing you to use
        unkey with any cloud provider effortlessly.
      </FeatureContent>
    </Feature>

    {/* <div
      ref={ref}
      className={cn(
        "rounded-[38px] border border-[rgba(255, 255, 255, 0.15)] bg-card text-card-foreground shadow-sm",
        className
      )}
      {...props}
    /> */}
  </div>
));
FeatureGrid.displayName = "FeatureGrid";

export { FeatureGrid };
