"use client";
import * as React from "react";
import { useSignUp } from "@clerk/nextjs";
import type { OAuthStrategy } from "@clerk/types";
import { Icons } from "@/components/ui/icons";
import { Button } from "@/components/ui/button";
import { useToast } from "@/components/ui/use-toast";

export function OAuthSignUp() {
  const [isLoading, setIsLoading] = React.useState<OAuthStrategy | null>(null);
  const { signUp, isLoaded: signupLoaded } = useSignUp();
  const { toast } = useToast();

  const oauthSignIn = async (provider: OAuthStrategy) => {
    if (!signupLoaded) {
      return null;
    }
    try {
      setIsLoading(provider);
      await signUp.authenticateWithRedirect({
        strategy: provider,
        redirectUrl: "/auth/sso-callback",
        redirectUrlComplete: "/onboarding",
      });
    } catch (cause) {
      console.error(cause);
      setIsLoading(null);
      toast({
        variant: "destructive",
        title: "Error",
        description: "Something went wrong, please try again.",
      });
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <Button
        variant="outline"
        className="bg-background"
        onClick={() => oauthSignIn("oauth_github")}
      >
        {isLoading === "oauth_github" ? (
          <Icons.spinner className="mr-2 h-4 w-4 animate-spin" />
        ) : (
          <Icons.gitHub className="mr-2 h-4 w-4" />
        )}
        Github
      </Button>
      <Button
        variant="outline"
        className="bg-background"
        onClick={() => oauthSignIn("oauth_google")}
      >
        {isLoading === "oauth_google" ? (
          <Icons.spinner className="mr-2 h-4 w-4 animate-spin" />
        ) : (
          <Icons.google className="mr-2 h-4 w-4" />
        )}
        Google
      </Button>
    </div>
  );
}
