"use client";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/Form";
import {
  Select,
  SelectItem,
  SelectGroup,
  SelectLabel,
  SelectValue,
  SelectContent,
  SelectTrigger,
  SelectSeparator,
} from "@/components/ui/select";
import { Loading } from "@/components/loading";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/use-toast";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useOrganization } from "@clerk/nextjs";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { MembershipRole } from "@clerk/types";
import { useState } from "react";
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

  const inviteUser = async ({ values }: { values: { email: string; role: MembershipRole } }) => {
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

  return (
    <div className="flex items-center justify-center w-full min-h-screen">
      <Card className="w-1/2">
        <CardHeader>
          <CardTitle>Add a member to your team</CardTitle>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form
              onSubmit={form.handleSubmit((values) => inviteUser({ values }))}
              className="flex flex-col space-y-4"
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
        </CardContent>
      </Card>
    </div>
  );
}
