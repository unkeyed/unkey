"use client";

import { Loading } from "@/components/dashboard/loading";
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
import {
  Form,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { trpc } from "@/lib/trpc/client";
import { parseTrpcError } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import type { Key } from "@unkey/db";
import ms from "ms";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

type Props = {
  trigger: React.ReactNode;
  currentKey: Key;
  apiId: string;
};

const EXPIRATION_OPTIONS = [
  { key: "now", value: "Now" },
  { key: "5m", value: "5 minutes" },
  { key: "30m", value: "30 minutes" },
  { key: "1h", value: "1 hour" },
  { key: "6h", value: "6 hours" },
  { key: "24h", value: "24 hours" },
  { key: "7d", value: "7 days" },
];

const formSchema = z.object({
  expiresIn: z.coerce.string(),
});

export const RerollKey: React.FC<Props> = ({ trigger, currentKey, apiId }: Props) => {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const { keyAuthId } = currentKey;
  const [newKeyId, setNewKeyId] = useState<string | null>();

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    reValidateMode: "onBlur",
    defaultValues: {
      expiresIn: "1h",
    },
  });

  const createKey = trpc.key.create.useMutation({
    onSuccess({ keyId }) {
      setNewKeyId(keyId);
    },
    onError(err) {
      console.error(err);
      const message = parseTrpcError(err);
      toast.error(message);
    },
  });

  const updateDeletedAt = trpc.key.update.deletedAt.useMutation({
    onSuccess() {
      toast.success("Rerolling completed.");
      router.push(`/apis/${apiId}/keys/${keyAuthId}/${newKeyId}`);
    },
    onError(err) {
      console.error(err);
      const message = parseTrpcError(err);
      toast.error(message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    toast.success("Rerolling in progress.");

    await createKey.mutateAsync({
      ...currentKey,
      keyAuthId: currentKey.keyAuthId,
      name: currentKey.name || undefined,
      environment: currentKey.environment || undefined,
      meta: currentKey.meta ? JSON.parse(currentKey.meta) : undefined,
      expires: currentKey.expires?.getTime() ?? undefined,
      remaining: currentKey.remaining ?? undefined,
    });

    const miliseconds = ms(values.expiresIn);
    const deletedAt = new Date(Date.now() + miliseconds);
    
    await updateDeletedAt.mutate({
      keyId: currentKey.id,
      deletedAt,
    });
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger>{trigger}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Regenerate API Key</DialogTitle>
          <DialogDescription>
            Rerolling creates a new identical key with the same configuration and automatically
            expires the current one. Make sure to replace it in your system before it expires.
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-8">
            <FormField
              control={form.control}
              name="expiresIn"
              render={({ field }) => (
                <FormItem className="">
                  <FormLabel>Expire previous key in:</FormLabel>
                  <Select onValueChange={field.onChange} defaultValue="1h" value={field.value}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {EXPIRATION_OPTIONS.map((item) => (
                        <SelectItem key={item.key} value={item.key}>
                          {item.value}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    Choose an optional delay period before the old key expires.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter>
              <Button type="submit">
                {createKey.isLoading || updateDeletedAt.isLoading ? (
                  <Loading className="w-4 h-4" />
                ) : (
                  "Reroll Key"
                )}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};
