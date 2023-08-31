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

export const UpdateUserEmail: React.FC = () => {
  const { toast } = useToast();
  const { user } = useUser();

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    defaultValues: {
      email: user?.emailAddresses.at(0)?.emailAddress ?? undefined,
    },
  });
  if (!user) {
    return null;
  }

  const isDisabled = form.formState.isLoading || !form.formState.isValid;
  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(async ({ email }) => {
          try {
            const emailResponse = await user.createEmailAddress({ email });
            const flow = emailResponse.createMagicLinkFlow();
            toast({
              title: "Confirm Email",
              description: `We have sent an email to ${email}, please confirm it by clicking the link in the email.`,
            });
            const magic = await flow.startMagicLinkFlow({ redirectUrl: "TODO" });
            if (magic.verification.status === "verified") {
              toast({
                title: "Success",
                description: "Workspace name updated",
              });
              user.reload();
              return;
            }
            toast({
              title: "Error",
              description: `Something went wrong: ${magic.verification.error?.message}`,
              variant: "alert",
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
              control={form.control}
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
              {form.formState.isLoading ? <Loading /> : "Save"}
            </Button>
          </CardFooter>
        </Card>
      </form>
    </Form>
  );
};
