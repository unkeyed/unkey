"use client";
import { Loading } from "@/components/dashboard/loading";
import { GitHub, Google } from "@/components/ui/icons";
import { toast } from "@/components/ui/toaster";
import type { OAuthStrategy } from "@/lib/auth/interface";
import * as React from "react";
import { OAuthButton } from "../oauth-button";
import { initiateOAuthSignIn } from "../actions";

export function OAuthSignUp() {
  const [isLoading, setIsLoading] = React.useState<OAuthStrategy | null>(null);
  const redirectUrlComplete = "/new";

  const oauthSignIn = async (provider: OAuthStrategy) => {
    try {
      setIsLoading(provider);
      const result = await initiateOAuthSignIn({ provider, redirectUrlComplete });
      if (result.error) {
        throw new Error(`OAuth error: ${result.error}`);
      }

      if (result.url) {
        window.location.assign(result.url);
      }
        
      } catch (err) {
        console.error(err);
        setIsLoading(null);
        toast.error((err as Error).message);
      }
  };

  return (
    <div className="flex flex-col gap-2">
      <OAuthButton onClick={() => oauthSignIn("github")}>
        {isLoading === "oauth_github" ? (
          <Loading className="w-6 h-6" />
        ) : (
          <GitHub className="w-6 h-6" />
        )}
        GitHub
      </OAuthButton>
      <OAuthButton onClick={() => oauthSignIn("google")}>
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
