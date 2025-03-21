"use client";

import { Loading } from "@/components/dashboard/loading";
import { GitHub, Google } from "@/components/ui/icons";
import { toast } from "@/components/ui/toaster";
import { signInViaOAuth } from "@/lib/auth/actions";
import type { OAuthStrategy } from "@/lib/auth/types";
import { useSearchParams } from "next/navigation";
import * as React from "react";
import { OAuthButton } from "../oauth-button";
import { LastUsed, useLastUsed } from "./last_used";

export const OAuthSignIn: React.FC = () => {
  const [isLoading, setIsLoading] = React.useState<OAuthStrategy | null>(null);
  const [lastUsed, setLastUsed] = useLastUsed();
  const searchParams = useSearchParams();
  const redirectUrlComplete = searchParams?.get("redirect") ?? "/apis";

  const oauthSignIn = async (provider: OAuthStrategy) => {
    try {
      setIsLoading(provider);
      setLastUsed(provider);

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
        {isLoading === "github" ? <Loading className="w-6 h-6" /> : <GitHub className="w-6 h-6" />}
        GitHub {lastUsed === "github" ? <LastUsed /> : null}
      </OAuthButton>
      <OAuthButton onClick={() => oauthSignIn("google")}>
        {isLoading === "google" ? <Loading className="w-6 h-6" /> : <Google className="w-6 h-6" />}
        Google {lastUsed === "google" ? <LastUsed /> : null}
      </OAuthButton>
    </div>
  );
};
