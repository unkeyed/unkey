"use client";
import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { FadeIn } from "@/components/landing/fade-in";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogFooter,
} from "@/components/ui/alert-dialog";
import { useAuth } from "@clerk/nextjs";
import { SearchX } from "lucide-react";
import { useRouter } from "next/navigation";
import * as React from "react";
import { EmailCode } from "../email-code";
import { EmailSignIn } from "../email-signin";
import { OAuthSignIn } from "../oauth-signin";

export default function AuthenticationPage() {
  const [verify, setVerify] = React.useState(false);
  const [showDialog, setShowDialog] = React.useState(false);
  const [email, setEmail] = React.useState("");
  const { isLoaded } = useAuth();
  const router = useRouter();
  if (!isLoaded) {
    return null;
  }
  return (
    <div className="mx-auto flex w-full flex-col justify-center space-y-6 px-6 sm:w-[500px] md:px-0">
      {!verify && !showDialog && (
        <>
          <div className="flex flex-col space-y-2 text-center">
            <h1 className="text-3xl font-semibold tracking-tight text-black">Sign In to Unkey</h1>
            <p className="text-md text-content-subtle">Enter your email below to sign in</p>
          </div>
          <div className="grid gap-6">
            <EmailSignIn
              verification={setVerify}
              dialog={setShowDialog}
              email={setEmail}
              emailValue={email}
            />

            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <span className="w-full border-t border-gray-300" />
              </div>
              <div className="relative flex justify-center text-xs uppercase">
                <span className="text-content-subtle bg-gray-50 px-2">Or continue with</span>
              </div>
            </div>
            <OAuthSignIn />
          </div>
          <div className="relative flex justify-center text-xs uppercase">
            <span className="text-content-subtle bg-gray-50 px-2">
              Not been here before? Just{" "}
              <a className="text-black" href="/auth/sign-up">
                Sign Up
              </a>
            </span>
          </div>
        </>
      )}
      {verify && (
        <FadeIn>
          <div className="flex flex-col space-y-2 text-center">
            <h1 className="text-3xl font-semibold tracking-tight">Enter your email code</h1>
            <p className="text-md text-content-subtle">We sent you a 6 digit code to your email</p>
            <EmailCode />
          </div>
        </FadeIn>
      )}
      <AlertDialog open={showDialog}>
        <AlertDialogContent>
          <EmptyPlaceholder className="min-h-full border-0">
            <EmptyPlaceholder.Icon>
              <SearchX />
            </EmptyPlaceholder.Icon>
            <EmptyPlaceholder.Title>No account found</EmptyPlaceholder.Title>

            <EmptyPlaceholder.Description>
              We didn't detect an account associated with{" "}
              <span className="font-semibold">{email}</span>. Did you mean to sign up?
            </EmptyPlaceholder.Description>
          </EmptyPlaceholder>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setShowDialog(false)}>No</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => router.push(`/auth/sign-up?email=${encodeURIComponent(email)}`)}
            >
              Sign up
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
