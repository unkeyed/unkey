"use client";
import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

export const CreateSecretForm: React.FC = () => {
  const formSchema = z.object({
    name: z.string(),
    value: z.string(),
    comment: z.string().optional(),
  });

  const router = useRouter();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
  });

  const create = trpc.secrets.create.useMutation({
    onSuccess() {
      toast.success("Secret created");
      router.refresh();
    },
    onError(err) {
      toast.error("An error occured", {
        description: err.message,
      });
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    create.mutate(values);
  }

  return (
    <Form {...form}>
      <form className="flex flex-col gap-4" onSubmit={form.handleSubmit(onSubmit)}>
        <Separator className="mt-4" />

        <div className="flex items-start w-full gap-4">
          <FormField
            control={form.control}
            name="name"
            render={({ field }) => (
              <FormItem className="w-full">
                <FormLabel>Name</FormLabel>
                <FormControl>
                  <Input {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="value"
            render={({ field }) => (
              <FormItem className="w-full">
                <FormLabel>Value</FormLabel>
                <FormControl>
                  <Input {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>

        <FormField
          control={form.control}
          name="comment"
          render={({ field }) => (
            <FormItem className="w-full">
              <FormLabel>Comment</FormLabel>
              <FormControl>
                <Input
                  {...field}
                  placeholder="Comments can be used to explain what this secret is used for"
                />
              </FormControl>

              <FormMessage />
            </FormItem>
          )}
        />

        <Separator className="" />

        <div className="w-full">
          <Button
            className="w-full"
            disabled={!form.formState.isValid || create.isLoading}
            type="submit"
            variant={create.isLoading || !form.formState.isValid ? "disabled" : "primary"}
          >
            {create.isLoading ? <Loading className="w-4 h-4" /> : "Encrypt"}
          </Button>
        </div>
      </form>
    </Form>
  );
};
