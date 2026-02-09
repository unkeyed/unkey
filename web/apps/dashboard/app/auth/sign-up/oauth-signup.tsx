"use client";

import { GitHub, Google } from "@/components/ui/icons";
import { signInWithSocial } from "@/lib/auth/better-auth-client";
import type { OAuthStrategy } from "@/lib/auth/types";
import { Loading, toast } from "@unkey/ui";
import * as React from "react";
import { OAuthButton } from "../oauth-button";

export function OAuthSignUp() {
  const [isLoading, setIsLoading] = React.useState<OAuthStrategy | null>(null);
  const [clientReady, setClientReady] = React.useState(false);
  const redirectUrlComplete = "/new";

  // Set clientReady to true after hydration is complete
  React.useEffect(() => {
    setClientReady(true);
  }, []);

  const oauthSignIn = async (provider: OAuthStrategy) => {
    try {
      setIsLoading(provider);
      // Better Auth client handles the redirect automatically
      await signInWithSocial(provider, redirectUrlComplete);
    } catch (_err) {
      toast.error("Failed to initiate login. Please try again.");
      setIsLoading(null);
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <OAuthButton onClick={() => oauthSignIn("github")}>
        {clientReady && isLoading === "github" ? (
          <Loading className="w-6 h-6" />
        ) : (
          <GitHub className="w-6 h-6" />
        )}
        GitHub
      </OAuthButton>
      <OAuthButton onClick={() => oauthSignIn("google")}>
        {clientReady && isLoading === "google" ? (
          <Loading className="w-6 h-6" />
        ) : (
          <Google className="w-6 h-6" />
        )}
        Google
      </OAuthButton>
    </div>
  );
}
