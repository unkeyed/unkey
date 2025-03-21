"use client";
import { Loading } from "@/components/dashboard/loading";
import { GitHub, Google } from "@/components/ui/icons";
import { toast } from "@/components/ui/toaster";
import { useSignIn } from "@clerk/nextjs";
import type { OAuthStrategy } from "@clerk/types";
import * as React from "react";
import { useIsClient } from "usehooks-ts";
import { OAuthButton } from "../oauth-button";
import { LastUsed, useLastUsed } from "./last_used";

export const OAuthSignIn: React.FC = () => {
  const isClient = useIsClient();
  const [isLoading, setIsLoading] = React.useState<OAuthStrategy | null>(null);
  const { signIn, isLoaded: signInLoaded } = useSignIn();
  const [lastUsed, setLastUsed] = useLastUsed();

  const oauthSignIn = async (provider: OAuthStrategy) => {
    if (!signInLoaded) {
      return null;
    }
    try {
      setIsLoading(provider);
      await signIn.authenticateWithRedirect({
        strategy: provider,
        redirectUrl: "/auth/sso-callback",
        redirectUrlComplete: "/apis",
      });
      setIsLoading(null);
      setLastUsed(provider === "oauth_google" ? "google" : "github");
    } catch (err) {
      console.error(err);
      setIsLoading(null);
      toast.error((err as Error).message);
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <OAuthButton
        disabled={isLoading === "oauth_github"}
        onClick={() => oauthSignIn("oauth_github")}
      >
        {isLoading === "oauth_github" ? (
          <Loading className="w-6 h-6" />
        ) : (
          <GitHub className="w-6 h-6" />
        )}
        GitHub {isClient && lastUsed === "github" ? <LastUsed /> : null}
      </OAuthButton>
      <OAuthButton
        disabled={isLoading === "oauth_google"}
        onClick={() => oauthSignIn("oauth_google")}
      >
        {isLoading === "oauth_google" ? (
          <Loading className="w-6 h-6" />
        ) : (
          <Google className="w-6 h-6" />
        )}
        Google {isClient && lastUsed === "google" ? <LastUsed /> : null}
      </OAuthButton>
    </div>
  );
};
