"use client";
import { Button } from "@/components/ui/button";
import { Github, Loader2 } from "lucide-react";
import { useFormStatus } from "react-dom";

export const SignInButton: React.FC = () => {
  const { pending } = useFormStatus();

  return (
    <Button
      variant="secondary"
      type="submit"
      disabled={pending}
      className="border border-gray-300"
      aria-disabled={pending}
    >
      {pending ? (
        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
      ) : (
        <Github className="mr-2 h-4 w-4" />
      )}{" "}
      Sign in with GitHub
    </Button>
  );
};
