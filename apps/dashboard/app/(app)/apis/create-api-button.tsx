"use client";
import { revalidate } from "@/app/actions";
import { Loading } from "@/components/dashboard/loading";
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
import { Button } from "@unkey/ui";
import { Plus } from "lucide-react";
import { useRouter } from "next/navigation";
import type React from "react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  name: z.string().trim().min(3, "Name must be at least 3 characters long").max(50),
});

type Props = {
  defaultOpen?: boolean;
};

export const CreateApiButton = ({
  defaultOpen,
  ...rest
}: React.ButtonHTMLAttributes<HTMLButtonElement> & Props) => {
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const [open, setOpen] = useState(defaultOpen ?? false);

  const create = trpc.api.create.useMutation({
    async onSuccess(res) {
      toast.success("Your API has been created");
      await revalidate("/apis");
      router.push(`/apis/${res.id}`);
    },
    onError(err) {
      console.error(err);
      console.info(err)
      toast.error(err.message);
    },
  });
  async function onSubmit(values: z.infer<typeof formSchema>) {
    create.mutate(values);
  }
  const router = useRouter();

  return (
    <>
      <Dialog open={open} onOpenChange={(o) => setOpen(o)}>
        <DialogTrigger asChild>
          <Button variant="primary" {...rest}>
            <Plus />
            Create New API
          </Button>
        </DialogTrigger>
        <DialogContent className="border-border w-11/12 max-sm: ">
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
                        placeholder="my-api"
                        {...field}
                        className=" dark:focus:border-gray-700"
                      />
                    </FormControl>
                    <FormDescription>
                      This is just a human readable name for you and not visible to anyone else
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <DialogFooter className="flex-row justify-end gap-2 pt-4 ">
                <Button
                  variant="primary"
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
