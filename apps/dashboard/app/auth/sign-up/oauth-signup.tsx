"use client";
import { Loading } from "@/components/dashboard/loading";
import { GitHub, Google } from "@/components/ui/icons";
import { toast } from "@/components/ui/toaster";
import { useSignUp } from "@clerk/nextjs";
import type { OAuthStrategy } from "@clerk/types";
import * as React from "react";
import { OAuthButton } from "../oauth-button";

export function OAuthSignUp() {
  const [isLoading, setIsLoading] = React.useState<OAuthStrategy | null>(null);
  const { signUp, isLoaded: signupLoaded } = useSignUp();

  const oauthSignIn = async (provider: OAuthStrategy) => {
    if (!signupLoaded) {
      return null;
    }
    try {
      setIsLoading(provider);
      await signUp.authenticateWithRedirect({
        strategy: provider,
        redirectUrl: "/auth/sso-callback",
        redirectUrlComplete: "/new",
      });
      setIsLoading(null);
    } catch (cause) {
      console.error(cause);
      setIsLoading(null);
      toast.error("Something went wrong, please try again.");
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <OAuthButton onClick={() => oauthSignIn("oauth_github")}>
        {isLoading === "oauth_github" ? (
          <Loading className="w-6 h-6" />
        ) : (
          <GitHub className="w-6 h-6" />
        )}
        GitHub
      </OAuthButton>
      <OAuthButton onClick={() => oauthSignIn("oauth_google")}>
        {isLoading === "oauth_google" ? (
          <Loading className="w-6 h-6" />
        ) : (
          <Google className="w-6 h-6" />
        )}
        Google
      </OAuthButton>
    </div>
  );
}
