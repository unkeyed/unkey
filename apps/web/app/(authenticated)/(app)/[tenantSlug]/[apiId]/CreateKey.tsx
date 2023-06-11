"use client";

import { toast } from "sonner";
import { useRouter } from "next/navigation";

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/Form";
import { Loading } from "@/components/loading";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  bytes: z.number().int().gte(1),
  prefix: z.string().max(8).optional(),
  ownerId: z.string().optional(),
  meta: z.record(z.unknown()).optional(),
});
type Props = {
  apiId: string;
};

export const CreateKeyButton: React.FC<Props> = ({ apiId }) => {
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      prefix: "api",
      bytes: 16,
    },
  });
  const router = useRouter();
  const create = trpc.key.create.useMutation({
    onSuccess() {
      toast.success("Key Created", {
        description: "Your Key has been created",
      });
      console.log("XXX");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error("Error", { description: err.message });
    },
  });
  async function onSubmit(values: z.infer<typeof formSchema>) {
    await create.mutateAsync({
      apiId,
      ...values,
    });
  }

  return (
    <>
      <Dialog>
        <DialogTrigger asChild>
          <Button>Create Key</Button>
        </DialogTrigger>

        <DialogContent>
          <DialogTitle>Create a new API</DialogTitle>

          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col space-y-4">
              <FormField
                control={form.control}
                name="prefix"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Prefix</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormDescription>
                      Using a prefix can make it easier for your users to distinguis between api
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="bytes"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Bytes</FormLabel>
                    <FormControl>
                      <Input type="number" {...field} />
                    </FormControl>
                    <FormDescription>How many bytes to use.</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="ownerId"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Owner</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormDescription>
                      This is the id of the user or tenant in your system, so you can identify users
                      from an API key.
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <DialogFooter className="justify-end">
                <Button type="submit">{create.isLoading ? <Loading /> : "Create"}</Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>
    </>
  );
};
