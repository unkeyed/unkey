"use client";
import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import { Icons } from "@/components/ui/icons";
import { toast } from "@/components/ui/toaster";
import { useSignUp } from "@clerk/nextjs";
import type { OAuthStrategy } from "@clerk/types";
import * as React from "react";

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
    } catch (cause) {
      console.error(cause);
      setIsLoading(null);
      toast.error("Something went wrong, please try again.");
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <Button
        variant="secondary"
        className="bg-background"
        onClick={() => oauthSignIn("oauth_github")}
      >
        {isLoading === "oauth_github" ? (
          <Loading className="w-4 h-4 mr-2" />
        ) : (
          <Icons.gitHub className="w-4 h-4 mr-2" />
        )}
        GitHub
      </Button>
      <Button
        variant="secondary"
        className="bg-background"
        onClick={() => oauthSignIn("oauth_google")}
      >
        {isLoading === "oauth_google" ? (
          <Loading className="w-4 h-4 mr-2" />
        ) : (
          <Icons.google className="w-4 h-4 mr-2" />
        )}
        Google
      </Button>
    </div>
  );
}
