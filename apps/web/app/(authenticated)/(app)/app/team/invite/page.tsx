"use client";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import {
  Select,
  SelectItem,
  SelectValue,
  SelectContent,
  SelectTrigger,
} from "@/components/ui/select";
import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/use-toast";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useOrganization } from "@clerk/nextjs";
import { MembershipRole } from "@clerk/types";
import { useState } from "react";
import { redirect } from "next/navigation";
const formSchema = z.object({
  email: z.string().email(),
  role: z.enum(["admin", "basic_member"], {
    required_error: "You need to select a role type.",
  }),
});

export default function TeamCreation() {
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });
  const [isLoading, setIsLoading] = useState(false);
  const { organization } = useOrganization();

  const { toast } = useToast();

  const inviteUser = async ({
    values,
  }: {
    values: { email: string; role: MembershipRole };
  }) => {
    setIsLoading(true);
    await organization
      ?.inviteMember({
        emailAddress: values.email,
        role: values.role,
      })
      .then(() => {
        toast({
          title: "Invite Sent",
          description: `We've sent an invitation to ${values.email}`,
          variant: "default",
        });
        setIsLoading(false);
        form.reset({
          email: "",
          role: "basic_member",
        });
      })
      .catch((err) => {
        setIsLoading(false);
        console.error(err);
        toast({
          title: "Error",
          description: "Error sending invite, please reach out to support",
          variant: "destructive",
        });
      });
  };
  if (!organization) {
    return redirect("/app/team/create");
  }

  return (
    <div className="flex flex-col gap-4 pt-4">
      <h1 className=" text-4xl font-semibold leading-none tracking-tight">Invite a new user</h1>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit((values) => inviteUser({ values }))}
          className="flex flex-col space-y-4 max-w-2xl"
        >
          <FormField
            control={form.control}
            name="email"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Email</FormLabel>
                <FormControl>
                  <Input {...field} />
                </FormControl>
                <FormDescription>
                  The user will be invited to accept the team invite by email.{" "}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="role"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Role</FormLabel>
                <FormControl>
                  <Select
                    onValueChange={field.onChange as (value: string) => void}
                    defaultValue={field.value}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Member" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="admin">Admin</SelectItem>
                      <SelectItem value="gh">Member</SelectItem>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormDescription>An admin can invite other users. </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className="mt-8">
            <Button
              disabled={isLoading || !form.formState.isValid}
              type="submit"
              className="w-full"
            >
              {isLoading ? <Loading /> : "Invite"}
            </Button>
          </div>
        </form>
      </Form>
    </div>
  );
}
