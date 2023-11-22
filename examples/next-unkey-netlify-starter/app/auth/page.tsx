"use client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/use-toast";
import { cn } from "@/lib/utils";
import { useEffect } from "react";
import { useFormState, useFormStatus } from "react-dom";
import { loginAction } from "../actions/actions";
const initialState = {
  message: "",
};
function SubmitButton() {
  const { pending } = useFormStatus();

  return (
    <Button
      type="submit"
      className={cn("h-9 px-4 w-full py-2", pending && "animate-pulse")}
      aria-disabled={pending}
    >
      Login with email
    </Button>
  );
}
const Login = () => {
  const { toast } = useToast();
  const [state, formAction] = useFormState(loginAction, initialState);
  useEffect(() => {
    if (state?.message && state.message.length > 0) {
      toast({ title: state.message });
    }
  }, [state, toast]);
  return (
    <form
      action={formAction}
      className="flex min-h-screen flex-col items-center justify-center p-24"
    >
      <h1 className="text-xl font-bold mb-2">Create an account</h1>
      <p className="text-base text-muted-foreground mb-5 ">
        Enter your email to sign up for an account
      </p>
      <div className="w-1/2">
        <Input className="my-2" type="email" />
        <SubmitButton />
      </div>
    </form>
  );
};

export default Login;
