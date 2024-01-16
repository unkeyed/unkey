"use client";
import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { ClerkError } from "@/lib/clerk";
import { useUser } from "@clerk/nextjs";
import { zodResolver } from "@hookform/resolvers/zod";
import React from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const validCharactersRegex = /^[a-zA-Z0-9-_]+$/;

const formSchema = z.object({
  username: z
    .string()
    .min(3)
    .refine((v) => validCharactersRegex.test(v), {
      message: "Username can only contain letters, numbers, dashes and underscores",
    }),
});

export const UpdateUserName: React.FC = () => {
  const { user } = useUser();

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    defaultValues: {
      username: user?.username ?? "",
    },
  });
  if (!user) {
    return (
      <EmptyPlaceholder className="min-h-[200px]">
        <Loading />
      </EmptyPlaceholder>
    );
  }

  const isDisabled = form.formState.isLoading || !form.formState.isValid;
  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(({ username }) => {
          user
            .update({ username })
            .then(() => {
              toast.success("Workspace name updated");
              user.reload();
            })
            .catch((err) => {
              toast.error(
                (err as ClerkError).errors.at(0)?.longMessage ??
                  "Sorry there was an error updating your username",
              );
            });
        })}
      >
        <Card>
          <CardHeader>
            <CardTitle>Username</CardTitle>
          </CardHeader>
          <CardContent>
            <FormField
              control={form.control}
              name="username"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Username</FormLabel>
                  <FormControl>
                    <Input {...field} className="max-w-sm" />
                  </FormControl>
                  <FormDescription>Update or create a username</FormDescription>
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
