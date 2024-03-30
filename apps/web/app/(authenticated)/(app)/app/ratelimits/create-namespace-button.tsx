"use client";
import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogTrigger } from "@/components/ui/dialog";
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
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus } from "lucide-react";
import { useRouter } from "next/navigation";
import type React from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  name: z.string().regex(/^[a-zA-Z0-9_\-\.]{3,50}$/),
});

export const CreateNamespaceButton = ({
  ...rest
}: React.ButtonHTMLAttributes<HTMLButtonElement>) => {
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const create = trpc.ratelimit.namespace.create.useMutation({
    onSuccess(res) {
      toast.success("Your Namespace has been created");
      router.refresh();
      router.push(`/app/ratelimits/${res.id}`);
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });
  async function onSubmit(values: z.infer<typeof formSchema>) {
    create.mutate(values);
  }
  const router = useRouter();

  return (
    <>
      <Dialog>
        <DialogTrigger asChild>
          <Button className="flex-row items-center gap-1 font-semibold " {...rest}>
            <Plus size={18} className="w-4 h-4 " />
            Create new namespace
          </Button>
        </DialogTrigger>
        <DialogContent className="w-11/12 max-sm: ">
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)}>
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Name</FormLabel>
                    <FormControl>
                      <Input
                        data-1p-ignore
                        placeholder="email.outbound"
                        {...field}
                        className=" dark:focus:border-gray-700"
                      />
                    </FormControl>
                    <FormDescription>
                      Alphanumeric, underscores, hyphens or periods are allowed.
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <DialogFooter className="flex-row justify-end gap-2 pt-4 ">
                <Button
                  disabled={create.isLoading || !form.formState.isValid}
                  className="mt-4 "
                  type="submit"
                >
                  {create.isLoading ? <Loading /> : "Create"}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>
    </>
  );
};
