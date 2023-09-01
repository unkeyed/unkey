"use client";
import { Button } from "@/components/ui/button";
import React from "react";
import { Loading } from "@/components/dashboard/loading";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/use-toast";
import { useUser } from "@clerk/nextjs";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  email: z.string().email(),
});

const verificationSchema = z.object({
  code: z.string().min(6).max(6),
});

export const UpdateUserEmail: React.FC = () => {
  const { toast } = useToast();
  const { user } = useUser();
  const [verification, setVerification] = React.useState(false);
  const emailForm = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    defaultValues: {
      email: user?.primaryEmailAddress?.emailAddress ?? "",
    },
  });
  const verificationForm = useForm<z.infer<typeof verificationSchema>>({
    resolver: zodResolver(verificationSchema),
    mode: "all",
  });

  if (!user) {
    return null;
  }

  const isDisabled = emailForm.formState.isLoading || !emailForm.formState.isValid;
  return (
    <>
      {!verification && (
        <Form {...emailForm}>
          <form
            onSubmit={emailForm.handleSubmit(async ({ email }) => {
              try {
                const emailResponse = await user.createEmailAddress({ email });
                await emailResponse
                  .prepareVerification({
                    strategy: "email_code",
                  })
                  .then(() => {
                    setVerification(true);
                  });

                toast({
                  title: "Confirm Email",
                  description: `We have sent an email to ${email}, please confirm it by entering the code we sent you.`,
                });
              } catch (e) {
                toast({
                  title: "Error",
                  description: (e as Error).message,
                  variant: "alert",
                });
              }
            })}
          >
            <Card>
              <CardHeader>
                <CardTitle>Email</CardTitle>
              </CardHeader>
              <CardContent>
                <FormField
                  control={emailForm.control}
                  name="email"
                  render={({ field }) => (
                    <FormItem>
                      <FormControl>
                        <Input {...field} className="max-w-sm" />
                      </FormControl>
                      <FormDescription>What's your email?</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </CardContent>
              <CardFooter className="justify-end">
                <Button
                  type="submit"
                  variant={isDisabled ? "disabled" : "primary"}
                  disabled={isDisabled}
                >
                  {emailForm.formState.isLoading ? <Loading /> : "Save"}
                </Button>
              </CardFooter>
            </Card>
          </form>
        </Form>
      )}
      {verification && (
        <Form {...verificationForm}>
          <form
            onSubmit={verificationForm.handleSubmit(async ({ code }) => {
              try {
                const enteredEmail = emailForm.getValues().email;
                const email = user.emailAddresses.find(
                  (email) => email.emailAddress === enteredEmail,
                );
                if (!email) {
                  throw new Error("Email not found");
                }
                const verify = await email.attemptVerification({ code });
                if (verify.verification.status === "verified") {
                  // finally set the email as primary
                  await user.update({
                    primaryEmailAddressId: email.id,
                  });
                  toast({
                    title: "Success",
                    description: `We have succesfully updated your primary email to ${email.emailAddress}`,
                  });
                  setVerification(false);
                } else {
                  toast({
                    title: "Error",
                    description: "Invalid verification code",
                    variant: "alert",
                  });
                }
              } catch (e) {
                toast({
                  title: "Error",
                  description: (e as Error).message,
                  variant: "alert",
                });
              }
            })}
          >
            <Card>
              <CardHeader>
                <CardTitle>Email</CardTitle>
              </CardHeader>
              <CardContent>
                <FormField
                  control={verificationForm.control}
                  name="code"
                  render={({ field }) => (
                    <FormItem>
                      <FormControl>
                        <Input {...field} className="max-w-sm" />
                      </FormControl>
                      <FormDescription>Enter your verification code</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </CardContent>
              <CardFooter className="justify-end">
                <Button
                  type="submit"
                  variant={isDisabled ? "disabled" : "primary"}
                  disabled={isDisabled}
                >
                  {verificationForm.formState.isLoading ? <Loading /> : "Verify"}
                </Button>
              </CardFooter>
            </Card>
          </form>
        </Form>
      )}
    </>
  );
};
