"use client";

import { PageLoading } from "@/components/dashboard/page-loading";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { Check, Connections, ExternalLink, TriangleWarning } from "@unkey/icons";
import { Alert, AlertDescription, AlertTitle, Button, Empty, SettingCard, toast } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { BillingNavbar } from "../billing-navbar";

export default function StripeConnectPage() {
  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const searchParams = useSearchParams();
  const [isConnecting, setIsConnecting] = useState(false);

  const {
    data: connectedAccount,
    isLoading,
    refetch,
  } = trpc.customerBilling.connect.getAccount.useQuery();

  const disconnectAccount = trpc.customerBilling.connect.disconnect.useMutation({
    onSuccess: () => {
      toast.success("Stripe account disconnected");
      refetch();
    },
    onError: (error) => {
      toast.error("Failed to disconnect", {
        description: error.message,
      });
    },
  });

  // Handle OAuth callback
  useEffect(() => {
    const success = searchParams.get("success");
    const error = searchParams.get("error");

    if (success === "true") {
      // Show toast and redirect after a brief delay to ensure single toast
      setTimeout(() => {
        toast.success("Stripe account connected successfully!");
        router.replace(`/${workspace.slug}/billing/connect`);
      }, 100);
    } else if (error) {
      toast.error("Failed to connect Stripe account", {
        description: error,
      });
      router.replace(`/${workspace.slug}/billing/connect`);
    }
  }, [searchParams, router, workspace.slug]);

  const handleConnect = () => {
    setIsConnecting(true);
    const clientId = process.env.NEXT_PUBLIC_STRIPE_CONNECT_CLIENT_ID;
    const redirectUri = `${window.location.origin}/api/billing/stripe-connect/callback`;
    const state = Buffer.from(
      JSON.stringify({
        workspaceId: workspace.id,
        timestamp: Date.now(),
      }),
    ).toString("base64");

    const params = new URLSearchParams({
      response_type: "code",
      client_id: clientId ?? "",
      scope: "read_write",
      redirect_uri: redirectUri,
      state,
    });

    const authUrl = `https://connect.stripe.com/oauth/authorize?${params.toString()}`;
    window.location.href = authUrl;
  };

  const handleDisconnect = () => {
    if (
      confirm(
        "Are you sure you want to disconnect your Stripe account? This will disable billing features.",
      )
    ) {
      disconnectAccount.mutate();
    }
  };

  // Check if billing beta is enabled
  if (!workspace.betaFeatures.billing) {
    return (
      <div>
        <BillingNavbar activePage={{ href: "connect", text: "Stripe Connect" }} />
        <div className="p-4">
          <Empty>
            <Empty.Icon />
            <Empty.Title>Customer Billing Not Enabled</Empty.Title>
            <Empty.Description>
              Customer billing is currently in beta. Contact support to enable this feature for your
              workspace.
            </Empty.Description>
          </Empty>
        </div>
      </div>
    );
  }

  if (isLoading) {
    return <PageLoading message="Loading connection status..." />;
  }

  return (
    <div>
      <BillingNavbar activePage={{ href: "connect", text: "Stripe Connect" }} />
      <div className="py-3 w-full flex items-center justify-center">
        <div className="w-[900px] flex flex-col justify-center items-center gap-5 mx-6 mt-4">
          {connectedAccount ? (
            <div className="w-full flex flex-col gap-4">
              <Alert variant="default">
                <Check className="h-4 w-4" />
                <AlertTitle>Stripe Account Connected</AlertTitle>
                <AlertDescription>
                  Your Stripe account is connected and ready to bill your end users.
                </AlertDescription>
              </Alert>

              <SettingCard
                title="Connected Account"
                description="Your Stripe account details"
                border="top"
              >
                <div className="flex flex-col gap-2 w-full">
                  <div className="text-sm">
                    <span className="text-gray-11">Account ID: </span>
                    <span className="font-mono">{connectedAccount.stripeAccountId}</span>
                  </div>
                  <div className="text-sm">
                    <span className="text-gray-11">Connected: </span>
                    <span>{new Date(connectedAccount.connectedAt).toLocaleDateString()}</span>
                  </div>
                </div>
              </SettingCard>

              <SettingCard
                title="Manage Connection"
                description="View your Stripe dashboard or disconnect your account"
                border="bottom"
              >
                <div className="flex gap-2 w-full justify-end">
                  <Button
                    variant="outline"
                    onClick={() => window.open("https://dashboard.stripe.com", "_blank")}
                  >
                    <ExternalLink className="w-4 h-4 mr-2" />
                    Stripe Dashboard
                  </Button>
                  <Button
                    variant="destructive"
                    onClick={handleDisconnect}
                    loading={disconnectAccount.isLoading}
                  >
                    <Connections className="w-4 h-4 mr-2" />
                    Disconnect
                  </Button>
                </div>
              </SettingCard>
            </div>
          ) : (
            <div className="w-full flex flex-col gap-4">
              <Alert variant="warn">
                <TriangleWarning className="h-4 w-4" />
                <AlertTitle>Stripe Account Not Connected</AlertTitle>
                <AlertDescription>
                  Connect your Stripe account to start billing your end users for API usage.
                </AlertDescription>
              </Alert>

              <SettingCard
                title="Connect Stripe"
                description="Link your Stripe account to enable customer billing. You'll be redirected to Stripe to authorize the connection."
                border="both"
              >
                <div className="flex justify-end w-full">
                  <Button variant="primary" onClick={handleConnect} loading={isConnecting}>
                    Connect with Stripe
                  </Button>
                </div>
              </SettingCard>

              <div className="text-sm text-gray-11 p-4 bg-gray-2 rounded-lg">
                <h4 className="font-medium mb-2">What happens when you connect?</h4>
                <ul className="list-disc list-inside space-y-1">
                  <li>You'll be redirected to Stripe to authorize the connection</li>
                  <li>Unkey will be able to create invoices in your Stripe account</li>
                  <li>Your end users will be billed directly through your Stripe account</li>
                  <li>
                    You maintain full control of your Stripe account and can disconnect at any time
                  </li>
                </ul>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
