"use client";

import { Loading } from "@/components/dashboard/loading";
import { GitHub, Google } from "@/components/ui/icons";
import { toast } from "@/components/ui/toaster";
import type { OAuthStrategy } from "@/lib/auth/types";
import { getBaseUrl } from "@/lib/utils";
import * as React from "react";
import { signInViaOAuth } from "../actions";
import { OAuthButton } from "../oauth-button";

export function OAuthSignUp() {
  const [isLoading, setIsLoading] = React.useState<OAuthStrategy | null>(null);
  const [clientReady, setClientReady] = React.useState(false);
  const redirectUrlComplete = "/new";
  const baseUrl = getBaseUrl();

  // Set clientReady to true after hydration is complete
  React.useEffect(() => {
    setClientReady(true);
  }, []);

  const oauthSignIn = async (provider: OAuthStrategy) => {
    try {
      setIsLoading(provider);
      const url = await signInViaOAuth({
        provider,
        redirectUrlComplete,
      });

      if (url) {
        window.location.assign(url);
      } else {
        throw new Error("Failed to get OAuth URL");
      }
    } catch (err) {
      console.error(err);
      toast.error("Failed to initiate login. Please try again.");
    } finally {
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
