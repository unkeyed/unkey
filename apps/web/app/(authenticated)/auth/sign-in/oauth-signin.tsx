"use client";
import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import { Icons } from "@/components/ui/icons";
import { useToast } from "@/components/ui/use-toast";
import { useSignIn } from "@clerk/nextjs";
import type { OAuthStrategy } from "@clerk/types";
import * as React from "react";

export function OAuthSignIn() {
  const [isLoading, setIsLoading] = React.useState<OAuthStrategy | null>(null);
  const { signIn, isLoaded: signInLoaded } = useSignIn();
  const { toast } = useToast();

  const oauthSignIn = async (provider: OAuthStrategy) => {
    if (!signInLoaded) {
      return null;
    }
    try {
      setIsLoading(provider);
      await signIn.authenticateWithRedirect({
        strategy: provider,
        redirectUrl: "/auth/sso-callback",
        redirectUrlComplete: "/app/apis",
      });
    } catch (cause) {
      console.error(cause);
      setIsLoading(null);
      toast({
        variant: "alert",
        title: "Error",
        description: "Something went wrong, please try again.",
      });
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <Button
        className="bg-gray-50 text-black border-gray-300 hover:bg-black hover:text-white"
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
        className="bg-gray-50 text-black border-gray-300 hover:bg-black hover:text-white"
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
