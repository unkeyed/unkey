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
import { useForm } from "react-hook-form";
import { z } from "zod";

type Props = {
  workspace: { plan: Workspace["plan"] };
  webhooks: Array<{ id: string; destination: string }>;
  keySpaces: Array<{ id: string; api: { name: string } }>;
};

export const CreateExportButton: React.FC<Props> = ({ webhooks, workspace, keySpaces }) => {
  const formSchema = z.object({
    interval: z.enum(
      workspace.plan === "free"
        ? ["15m", "1h", "24h"]
        : ["1m", "5m", "15m", "30m", "1h", "6h", "12h", "24h"],
    ),
    webhookId: z.string(),
    keySpaceId: z.string(),
  });

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const create = trpc.usageReporter.create.useMutation({
    onSuccess(res) {
      toast.success("Your webhook has been created");
      router.refresh();
      router.push(`/webhooks/${res.id}`);
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });
  async function onSubmit(values: z.infer<typeof formSchema>) {
    create.mutate({
      webhookId: values.webhookId,
      keySpaceId: values.keySpaceId,
      interval: ms(values.interval),
    });
  }
  const router = useRouter();

  return (
    <>
      <Dialog>
        <DialogTrigger asChild>
          <Button className="flex-row items-center gap-1 font-semibold ">
            <Plus size={18} className="w-4 h-4 " />
            Create new reporter
          </Button>
        </DialogTrigger>
        <DialogContent className="w-11/12 max-sm: ">
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)}>
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
                    <FormLabel>KeySpace</FormLabel>
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
                name="webhookId"
                render={({ field }) => (
                  <FormItem className="">
                    <FormLabel>Webhook</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {webhooks.map((wh) => (
                          <SelectItem key={wh.id} value={wh.id}>
                            {wh.destination}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
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
