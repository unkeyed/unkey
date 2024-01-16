"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Loading } from "@/components/dashboard/loading";
import { VisibleButton } from "@/components/dashboard/visible-button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Code } from "@/components/ui/code";
import { Separator } from "@/components/ui/separator";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { AlertCircle, KeyRound, Lock } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

type Steps =
  | {
      step: "CREATE_ROOT_KEY";
      key?: never;
      rootKey?: never;
    }
  | {
      step: "CREATE_KEY";
      key?: never;
      rootKey: string;
    }
  | {
      step: "VERIFY_KEY";
      key?: string;
      rootKey?: never;
    };

type Props = {
  apiId: string;
};

export const Keys: React.FC<Props> = ({ apiId }) => {
  const [step, setStep] = useState<Steps>({ step: "CREATE_ROOT_KEY" });
  const rootKey = trpc.key.createInternalRootKey.useMutation({
    onSuccess(res) {
      setStep({ step: "CREATE_KEY", rootKey: res.key });
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });
  const key = trpc.key.create.useMutation({
    onSuccess(res) {
      setStep({ step: "VERIFY_KEY", key: res.key });
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  const [showKey, setShowKey] = useState(false);
  const [showKeyInSnippet, setShowKeyInSnippet] = useState(false);

  const createKeySnippet = `curl -XPOST '${
    process.env.NEXT_PUBLIC_UNKEY_API_URL ?? "https://api.unkey.dev"
  }/v1/keys.createKey' \\
  -H 'Authorization: Bearer ${rootKey.data?.key}' \\
  -H 'Content-Type: application/json' \\
  -d '{
    "apiId": "${apiId}"
  }'
  `;

  const verifyKeySnippet = `curl -XPOST '${
    process.env.NEXT_PUBLIC_UNKEY_API_URL ?? "https://api.unkey.dev"
  }/v1/keys.verifyKey' \\
  -H 'Content-Type: application/json' \\
  -d '{
    "key": "${key.data?.key ?? "<YOUR_KEY>"}"
  }'
  `;

  function maskKey(key: string): string {
    if (key.length === 0) {
      return "";
    }
    const split = key.split("_");
    if (split.length === 1) {
      return "*".repeat(split.at(0)!.length);
    }
    return `${split.at(0)}_${"*".repeat(split.at(1)!.length)}`;
  }
  function AsideContent() {
    return (
      <div>
        <div className="space-y-2">
          <div className="bg-primary/5 inline-flex items-center justify-center rounded-full border p-4">
            <Lock className="text-primary h-6 w-6" />
          </div>
          <h4 className="text-lg font-medium">Root Keys</h4>
          <p className="text-content-subtle text-sm">
            Root keys create resources such as keys or APIs on Unkey. You should never give this to
            your users.
          </p>
        </div>
        <div className="space-y-2 max-sm:mt-4">
          <div className="bg-primary/5 inline-flex items-center justify-center rounded-full border p-4">
            <KeyRound className="text-primary h-6 w-6" />
          </div>
          <h4 className="text-lg font-medium">Regular Keys</h4>
          <p className="text-content-subtle text-sm">
            Regular API keys are used to authenticate your users. You can use your root key to
            create regular API keys and give them to your users.
          </p>
        </div>
      </div>
    );
  }
  return (
    <div className="flex items-start justify-between gap-16 ">
      <main className="max-sm:w-full md:w-3/4">
        <aside className="mb-4 w-full md:hidden">
          <AsideContent />
        </aside>
        {step.step === "CREATE_ROOT_KEY" ? (
          <EmptyPlaceholder>
            <EmptyPlaceholder.Description>
              Let's begin by creating a root key
            </EmptyPlaceholder.Description>

            <Button disabled={rootKey.isLoading} onClick={() => rootKey.mutate({ roles: ["*"] })}>
              {rootKey.isLoading ? <Loading /> : "Create Root Key"}
            </Button>
          </EmptyPlaceholder>
        ) : step.step === "CREATE_KEY" ? (
          <Card>
            <CardHeader>
              <CardTitle className="mb-4">Your root key</CardTitle>
              <CardDescription>
                <Alert>
                  <AlertCircle className="h-4 w-4" />
                  <AlertTitle>This key is only shown once and can not be recovered </AlertTitle>
                  <AlertDescription>
                    Please store it somewhere safe for future use.
                  </AlertDescription>
                </Alert>
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Code className="my-8 flex w-full items-center justify-between gap-4 max-sm:overflow-hidden max-sm:text-[9px]">
                {showKey ? step.rootKey : maskKey(step.rootKey)}
                <div className="flex items-start justify-between max-sm:absolute max-sm:right-16  md:gap-4">
                  <VisibleButton isVisible={showKey} setIsVisible={setShowKey} />
                  <CopyButton value={step.rootKey} />
                </div>
              </Code>

              <Separator className="my-8" />

              <h2 className="mt-2 text-xl font-medium">Try it out</h2>
              <p className="mt-2 text-gray-500">
                Use your new root key to create a new API key for your users:
              </p>
              <Code className="my-8 flex w-full items-start justify-between md:gap-4 ">
                <div className=" mt-10 overflow-hidden max-sm:text-[8px] md:text-xs">
                  {showKeyInSnippet
                    ? createKeySnippet
                    : createKeySnippet.replace(step.rootKey, maskKey(step.rootKey))}
                </div>
                <div className="flex items-start justify-between max-sm:absolute max-sm:right-16 md:gap-4">
                  <VisibleButton isVisible={showKeyInSnippet} setIsVisible={setShowKeyInSnippet} />
                  <CopyButton value={createKeySnippet} />
                </div>
              </Code>
            </CardContent>
            <CardFooter className="justify-between">
              <Button
                size="sm"
                variant="link"
                disabled={key.isLoading}
                onClick={() => key.mutate({ apiId })}
              >
                {key.isLoading ? <Loading /> : "Or click here to create a key"}
              </Button>
              <Button
                className="whitespace-nowrap max-sm:text-xs"
                size="sm"
                onClick={() => {
                  setStep({ step: "VERIFY_KEY" });
                }}
              >
                I have created a key
              </Button>
            </CardFooter>
          </Card>
        ) : step.step === "VERIFY_KEY" ? (
          <Card>
            <CardHeader>
              <CardTitle>Verify a key</CardTitle>
              <CardDescription>Use the key you created and verify it.</CardDescription>
            </CardHeader>
            <CardContent>
              {step.key ? (
                <Code className="my-8 flex w-full items-center justify-between gap-4 max-sm:text-[9px]">
                  {showKey ? step.key : maskKey(step.key)}
                  <div className="flex items-start justify-between gap-4">
                    <VisibleButton isVisible={showKey} setIsVisible={setShowKey} />
                    <CopyButton value={step.key} />
                  </div>
                </Code>
              ) : null}

              <Code className="my-8 flex w-full items-start justify-between gap-4 max-sm:text-[6px]">
                <div className=" mt-10 overflow-hidden max-sm:text-[8px] md:text-xs">
                  {step.key
                    ? showKeyInSnippet
                      ? verifyKeySnippet
                      : verifyKeySnippet.replace(step.key, maskKey(step.key))
                    : verifyKeySnippet}
                </div>
                <div className="flex items-start justify-between gap-4  max-sm:absolute max-sm:right-16">
                  {step.key ? (
                    <VisibleButton
                      isVisible={showKeyInSnippet}
                      setIsVisible={setShowKeyInSnippet}
                    />
                  ) : null}
                  <CopyButton value={verifyKeySnippet} />
                </div>
              </Code>
            </CardContent>
            <CardFooter className="justify-between">
              <Link href="https://unkey.dev/docs" target="_blank">
                <Button size="sm" variant="link">
                  Read more
                </Button>
              </Link>
              <Link href="/app">
                <Button size="sm">Let's go</Button>
              </Link>
            </CardFooter>
          </Card>
        ) : null}
      </main>
      <aside className="w-1/4 flex-col items-start justify-center space-y-16 max-md:hidden md:flex ">
        <AsideContent />
      </aside>
    </div>
  );
};
