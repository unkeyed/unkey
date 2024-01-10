"use client";

import { notFound, useSearchParams } from "next/navigation";
import { useState } from "react";

import { CheckCircle, Fingerprint, Loader2, PartyPopper, XCircle } from "lucide-react";
import toast, { Toaster } from "react-hot-toast";

import { Button } from "@/components/ui/button";
import { useUser } from "@clerk/nextjs";

function CodeCharacter({ char }: { char: string }) {
  return <div className="p-2 lg:p-4 font-mono text-xl lg:text-4xl rounded bg-gray-900">{char}</div>;
}

function Cancelled() {
  return (
    <div className="w-full min-h-screen flex items-center pt-[250px] px-4 flex-col">
      <Toaster />
      <div className="flex pt-10">
        <div className="flex justify-center items-center pr-10">
          <XCircle className="text-gray-100" />
        </div>
        <div className="flex-col">
          <h1 className="text-lg text-gray-100">Login cancelled</h1>
          <p className="text-sm text-gray-500">You can return to your CLI.</p>
        </div>
      </div>
    </div>
  );
}

function Success() {
  return (
    <div className="w-full min-h-screen flex items-center pt-[250px] px-4 flex-col">
      <Toaster />
      <div className="flex pt-10">
        <div className="flex justify-center items-center pr-10">
          <PartyPopper className="text-gray-100" />
        </div>
        <div className="flex-col">
          <h1 className="text-lg text-gray-100">Login successful!</h1>
          <p className="text-sm text-gray-500">You can return to your CLI.</p>
        </div>
      </div>
    </div>
  );
}

export default function Page() {
  const [loading, setLoading] = useState(false);
  const [cancelled, setCancelled] = useState(false);
  const [success, setSuccess] = useState(false);

  const searchParams = useSearchParams();

  const code = searchParams.get("code");
  const _redirect = searchParams.get("redirect");

  async function verify(opts: {
    code: string | null;
    redirect: string | null;
  }) {
    setLoading(true);
    try {
      const req = await fetch("/api/unkey", {
        method: "POST",
        body: JSON.stringify(opts),
        headers: {
          "Content-Type": "application/json",
        },
      });

      if (!req.ok) {
        throw new Error(`HTTP error! status: ${req.status}`);
      }

      const res = await req.json();

      try {
        const redirectUrl = new URL(res.redirect);
        redirectUrl.searchParams.append("code", res.code);
        redirectUrl.searchParams.append("key", res.key);

        await fetch(redirectUrl.toString());

        setLoading(false);
        setSuccess(true);
      } catch (_error) {
        console.error(_error);
        setLoading(false);
        toast.error("Error redirecting back to local CLI. Is your CLI running?");
      }
    } catch (_error) {
      setLoading(false);
      toast.error("Error creating Unkey API key.");
    }
  }

  async function cancel() {
    try {
      setLoading(true);
      const redirectUrl = new URL(_redirect as string);
      redirectUrl.searchParams.append("cancelled", "true");
      await fetch(redirectUrl.toString());
      setLoading(false);
      setCancelled(true);
    } catch (_error) {
      setLoading(false);
      toast.error("Error cancelling login. Is your local CLI running?");
    }
  }

  const { user } = useUser();

  if (!code || !_redirect) {
    return notFound();
  }

  const opts = { code, redirect: _redirect, id: user?.id };

  if (cancelled) {
    return <Cancelled />;
  }

  if (success) {
    return <Success />;
  }

  return (
    <div className="w-full min-h-screen flex items-center pt-[250px] px-4 flex-col">
      <Toaster />
      <div className="flex flex-col">
        <div className="flex ">
          <div className="flex justify-center items-center pr-4">
            <Fingerprint className="text-gray-100" />
          </div>
          <div className="flex-col">
            <h1 className="text-lg text-gray-100">Device confirmation</h1>
            <p className="text-sm text-gray-500">
              Please confirm this is the code shown in your terminal
            </p>
          </div>
        </div>
        <div>
          <div className="grid grid-flow-col gap-1 pt-6 leading-none lg:gap-3 auto-cols-auto">
            {code?.split("").map((char, i) => (
              <CodeCharacter char={char} key={`${char}-${i}`} />
            ))}
          </div>
          <div className="flex justify-center pt-6">
            <div className="flex items-center">
              <Button
                variant="default"
                className="mr-2"
                onClick={() => verify(opts)}
                disabled={loading}
              >
                {loading ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <CheckCircle className="mr-2 h-4 w-4" />
                )}
                Confirm code
              </Button>
              <Button variant="outline" onClick={() => cancel()}>
                <XCircle className="mr-2 h-4 w-4" />
                Cancel
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
