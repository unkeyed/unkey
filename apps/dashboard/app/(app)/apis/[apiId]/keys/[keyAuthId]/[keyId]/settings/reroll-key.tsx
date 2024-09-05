"use client";

import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Form,
  FormDescription,
  FormField,
  FormItem, FormMessage
} from "@/components/ui/form";
import { Label } from "@/components/ui/label";
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
  apiId: string;
  apiKey: Key & {
    roles: []
  };
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

export const RerollKey: React.FC<Props> = ({ apiKey, apiId }: Props) => {
  const router = useRouter();
  const [newKeyId, setNewKeyId] = useState<string | null>();

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    reValidateMode: "onBlur",
    defaultValues: {
      expiresIn: "1h",
    },
  });

  const createKey = trpc.key.create.useMutation({
    onMutate() {
      toast.success("Rerolling Key");
    },
    onSuccess({ keyId }) {
      setNewKeyId(keyId);
    },
    onError(err) {
      console.error(err);
      const message = parseTrpcError(err);
      toast.error(message);
    },
  });

  const updateNewKey = trpc.rbac.connectRoleToKey.useMutation({
    onError(err) {
      console.error(err);
      const message = parseTrpcError(err);
      toast.error(message);
    },
  });

  const updateDeletedAt = trpc.key.update.deletedAt.useMutation({
    onSuccess() {
      toast.success("Key Rerolled.");
      router.push(`/apis/${apiId}/keys/${apiKey.keyAuthId}/${newKeyId}/settings`);
    },
    onError(err) {
      console.error(err);
      const message = parseTrpcError(err);
      toast.error(message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    const ratelimit = apiKey.ratelimitLimit ? {
      async: apiKey.ratelimitAsync ?? false,
      duration: apiKey.ratelimitDuration ?? 0,
      limit: apiKey.ratelimitLimit ?? 0,
    } : undefined;

    const refill = apiKey.refillInterval ? {
      interval: apiKey.refillInterval ?? "daily",
      amount: apiKey.refillAmount ?? 0,
    } : undefined;
    
    const newKey = await createKey.mutateAsync({
      ...apiKey,
      keyAuthId: apiKey.keyAuthId,
      name: apiKey.name || undefined,
      environment: apiKey.environment || undefined,
      meta: apiKey.meta ? JSON.parse(apiKey.meta) : undefined,
      expires: apiKey.expires?.getTime() ?? undefined,
      remaining: apiKey.remaining ?? undefined,
      identityId: apiKey.identityId ?? undefined,
      ratelimit,
      refill,
    });

    apiKey.roles.forEach(async (role: { roleId : string }) => {
      await updateNewKey.mutateAsync({ roleId: role.roleId, keyId: newKey.keyId})
    });

    const miliseconds = ms(values.expiresIn);
    const deletedAt = new Date(Date.now() + miliseconds);
    
    await updateDeletedAt.mutate({
      keyId: apiKey.id,
      deletedAt,
    });
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-8">
        <Card className="relative border-alert">
          <CardHeader>
            <CardTitle>Reroll Key</CardTitle>
            <CardDescription>
              Rerolling creates a new identical key with the same configuration and automatically
              expires the current one. Make sure to replace it in your system before it expires.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex justify-between item-center">
            <div className="flex flex-col w-full space-y-2">
              <Label htmlFor="expiresIn">Expire old key in:</Label>
              <FormField
                control={form.control}
                name="expiresIn"
                render={({ field }) => (
                  <FormItem className="max-w-sm">
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
            </div>
          </CardContent>
          <CardFooter className="items-center justify-end gap-4">
            <Button type="submit" variant="alert">
              {createKey.isLoading || updateDeletedAt.isLoading ? (
                <Loading className="w-4 h-4" />
              ) : (
                "Reroll Key"
              )}
            </Button>
          </CardFooter>
        </Card>
      </form>
    </Form>
  );
};
