"use client";

import { Loading } from "@/components/dashboard/loading";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { PostHogEvent } from "@/providers/PostHogProvider";
import { type Workspace } from "@unkey/db";
import { AlertCircle, AlertTriangle } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import React, { useState } from "react";
type Props = {
  trigger: React.ReactNode;
  workspace: Workspace;
};
const tiers = {
  free: {
    name: "Free Tier",
    id: "free",
    href: "/app",
    price: 0,
    description: "Everything you need to start your next API!",
    buttonText: "Free",
    features: [
      "100 Monthly Active Keys",
      "2500 Successful Verifications per month",
      "Unlimited APIs",
      "7 Days Analytics Retention",
    ],
    footnotes: [],
  },
  pro: {
    name: "Pro Tier",
    id: "paid",
    href: "/app",
    price: 25,
    description: "For those with teams and more demanding needs",
    buttonText: "Pro",
    features: [
      "250 Monthly Active keys included *",
      "150,000 Successful Verifications included **",
      "Unlimited APIs",
      "Workspaces with team members",
      "90 Days Analytics Retention",
      "90 Days Audit Log Retention",
    ],
    footnotes: [
      " *  Additional active keys are billed at $0.10",
      " ** Additional verifications are billed at $10 per 100,000",
    ],
  },
  custom: {
    name: "Custom",
    id: "enterprise",
    href: "https://cal.com/team/unkey/unkey-chat",
    price: "Let's talk",
    description: "We offer custom pricing for those with volume needs",
    buttonText: "Schedule a call",
    features: [
      "Custom Verification Limits",
      "Custom Active Key Limits",
      "Pricing based on your needs",
      "Custom Analytics Retention",
      "Dedicated support contract",
      "Whitelist IP per API",
    ],
    footnotes: [],
  },
};
export const ChangePlan: React.FC<Props> = ({ workspace, trigger }) => {
  const router = useRouter();
  const [open, setOpen] = useState(false);

  const changePlan = trpc.workspace.changePlan.useMutation({
    onSuccess: (_data, variables, _context) => {
      toast.success("Your plan has been changed");
      PostHogEvent({
        name: "plan_changed",
        properties: { plan: variables.plan, workspace: variables.workspaceId },
      });
      router.refresh();
      setOpen(false);
    },
    onError: (error) => {
      toast.error(error.message);
      setOpen(false);
    },
  });
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-[425px] md:min-w-fit">
        <DialogHeader>
          <DialogTitle>Change plan</DialogTitle>
          <DialogDescription className="w-full">
            <p>
              You are currently on the{" "}
              <span className="font-bold capitalize">{workspace.plan}</span> plan.
            </p>
            <div className="flex flex-col mt-10 gap-y-6 sm:gap-x-6 md:flex-col lg:flex-row">
              {(["free", "pro", "custom"] as const).map((tier) => (
                <div
                  key={tiers[tier].id}
                  className={
                    "ring-1 ring-gray-200 flex w-full flex-col justify-between rounded-3xl bg-white p-8 shadow-lg lg:w-1/3 xl:p-10"
                  }
                >
                  <div className="flex items-center justify-between gap-x-4">
                    <h2
                      id={tiers[tier].id}
                      className={"text-gray-900 text-2xl font-semibold leading-8"}
                    >
                      {tiers[tier].name}
                    </h2>
                  </div>
                  <p className="mt-4 min-h-[3rem] text-sm leading-6 text-gray-600 tex">
                    {tiers[tier].description}
                  </p>
                  <p className="flex items-center mx-auto mt-6 gap-x-1">
                    {typeof tiers[tier].price === "number" ? (
                      <>
                        <span className="text-4xl font-bold tracking-tight text-center text-gray-900">
                          {`$${tiers[tier].price}`}
                        </span>
                        <span className="text-sm font-semibold leading-6 text-gray-600 mx-autotext-center">
                          {"/month"}
                        </span>
                      </>
                    ) : (
                      <span className="mx-auto text-4xl font-bold tracking-tight text-center text-gray-900">
                        {tiers[tier].price}
                      </span>
                    )}
                  </p>

                  <div className="flex flex-col justify-between grow">
                    <ul className="mt-8 space-y-3 text-sm leading-6 text-gray-600 xl:mt-10">
                      {tiers[tier].features.map((feature) => (
                        <li key={feature} className="flex gap-x-3">
                          <svg
                            xmlns="http://www.w3.org/2000/svg"
                            viewBox="0 0 24 24"
                            className="flex-none w-5 h-6 text-gray-700"
                            aria-hidden="true"
                          >
                            <path
                              fill="currentColor"
                              fillRule="evenodd"
                              d="M19.916 4.626a.75.75 0 0 1 .208 1.04l-9 13.5a.75.75 0 0 1-1.154.114l-6-6a.75.75 0 0 1 1.06-1.06l5.353 5.353l8.493-12.739a.75.75 0 0 1 1.04-.208Z"
                              clipRule="evenodd"
                            />
                          </svg>

                          {feature}
                        </li>
                      ))}
                    </ul>
                    {tiers[tier].footnotes && (
                      <ul className="mt-6 mb-8">
                        {tiers[tier].footnotes.map((footnote, i) => (
                          <li key={`note-${i}`} className="flex text-xs text-gray-600 gap-x-3">
                            {footnote}
                          </li>
                        ))}
                      </ul>
                    )}
                    {tiers[tier].id === "enterprise" ? (
                      <Link href="mailto:support@unkey.dev">
                        <Button
                          className="col-span-1 w-full"
                          variant={workspace.plan === "enterprise" ? "disabled" : "secondary"}
                          disabled={workspace.plan === "enterprise"}
                        >
                          Schedule a Call
                        </Button>
                      </Link>
                    ) : (
                      <Dialog>
                        <DialogTrigger asChild disabled={workspace.plan === tiers[tier].id}>
                          <Button
                            className="col-span-1"
                            variant={workspace.plan === tiers[tier].id ? "disabled" : "secondary"}
                          >
                            {changePlan.isLoading ? <Loading /> : tiers[tier].buttonText}
                          </Button>
                        </DialogTrigger>

                        <DialogContent className="border-[#b80f07]">
                          <DialogHeader className="mx-auto">
                            <AlertTriangle />
                          </DialogHeader>

                          <Alert variant="alert">
                            <AlertTitle>Warning</AlertTitle>
                            <AlertDescription>
                              You are about to switch to our Pro plan. Please note there is a 24
                              hour pause before you can switch plans again.
                            </AlertDescription>
                          </Alert>

                          <DialogFooter className="justify-end">
                            <Button
                              className="col-span-1"
                              variant="outline"
                              onClick={() => setOpen(false)}
                            >
                              Cancel
                            </Button>
                            <Button
                              className="col-span-1"
                              variant={"primary"}
                              disabled={workspace.plan === tiers[tier].id}
                              onClick={() =>
                                changePlan.mutateAsync({
                                  workspaceId: workspace.id,
                                  plan: tiers[tier].id === "free" ? "free" : "pro",
                                })
                              }
                            >
                              {changePlan.isLoading ? <Loading /> : "Switch"}
                            </Button>
                          </DialogFooter>
                        </DialogContent>
                      </Dialog>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </DialogDescription>
          <DialogClose />
        </DialogHeader>
      </DialogContent>
    </Dialog>
  );
};
