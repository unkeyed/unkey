"use client";
import { Button } from "@/components/ui/button";
import { Github } from "lucide-react";

export const SignInButton: React.FC = () => {
  return (
    <Button variant="secondary" type="submit" className="border border-gray-300">
      <Github className="mr-2 h-4 w-4" />
      Sign in with GitHub
    </Button>
  );
};
