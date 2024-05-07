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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import type { Workspace } from "@unkey/db";
import { Plus } from "lucide-react";
import ms from "ms";
import { useRouter } from "next/navigation";
import type React from "react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

type Props = {
  workspace: { plan: Workspace["plan"] };
  keySpaces: Array<{ id: string; api: { name: string } }>;
};

export const CreateNewMonitorButton: React.FC<Props> = ({ workspace, keySpaces }) => {
  const [isOpen, setOpen] = useState(false);
  const formSchema = z.object({
    interval: z.enum(
      workspace.plan === "free"
        ? ["15m", "1h", "24h"]
        : ["1m", "5m", "15m", "30m", "1h", "6h", "12h", "24h"],
    ),
    webhookUrl: z.string().url(),
    keySpaceId: z.string(),
  });

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const create = trpc.monitor.verification.create.useMutation({
    onSuccess(_res) {
      toast.success("Your webhook has been created");
      setOpen(false);
      router.refresh();
      router.push("/webhooks");
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });
  async function onSubmit(values: z.infer<typeof formSchema>) {
    create.mutate({
      webhookUrl: values.webhookUrl,
      keySpaceId: values.keySpaceId,
      interval: ms(values.interval),
    });
  }
  const router = useRouter();

  return (
    <>
      <Dialog open={isOpen} onOpenChange={setOpen}>
        <DialogTrigger asChild>
          <Button className="flex-row items-center gap-1 font-semibold ">
            <Plus size={18} className="w-4 h-4 " />
            Create new reporter
          </Button>
        </DialogTrigger>
        <DialogContent>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col space-y-4">
              <FormField
                control={form.control}
                name="interval"
                render={({ field }) => (
                  <FormItem className="">
                    <FormLabel>Interval</FormLabel>
                    <Select onValueChange={field.onChange} defaultValue="1h" value={field.value}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {formSchema.shape.interval._def.values.map((value) => (
                          <SelectItem key={value} value={value}>
                            {ms(ms(value), { long: true })}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="keySpaceId"
                render={({ field }) => (
                  <FormItem className="">
                    <FormLabel>API</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {keySpaces.map((ks) => (
                          <SelectItem key={ks.id} value={ks.id}>
                            {ks.api.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="webhookUrl"
                render={({ field }) => (
                  <FormItem className="">
                    <FormLabel>Webhook Destination</FormLabel>
                    <Input type="url" {...field} />
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
