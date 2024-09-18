"use client";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Form, FormDescription, FormField, FormItem, FormMessage } from "@/components/ui/form";
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
import type { EncryptedKey, Key, Permission, Role } from "@unkey/db";
import ms from "ms";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { RerollConfirmationDialog } from "./reroll-confirmation-dialog";
import { RerollNewKeyDialog } from "./reroll-new-key-dialog";

type Props = {
  apiId: string;
  apiKey: Key & {
    roles: Role[];
    permissions: Permission[];
    encrypted: EncryptedKey;
  };
  lastUsed: number;
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

export const RerollKey: React.FC<Props> = ({ apiKey, apiId, lastUsed }: Props) => {
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
    onError(err) {
      console.error(err);
      const message = parseTrpcError(err);
      toast.error(message);
    },
  });

  const copyRolesToNewKey = trpc.rbac.connectRoleToKey.useMutation({
    onError(err) {
      console.error(err);
      const message = parseTrpcError(err);
      toast.error(message);
    },
  });

  const copyPermissionsToNewKey = trpc.rbac.addPermissionToKey.useMutation({
    onError(err) {
      console.error(err);
      const message = parseTrpcError(err);
      toast.error(message);
    },
  });

  const copyEncryptedToNewKey = trpc.key.update.encrypted.useMutation({
    onError(err) {
      console.error(err);
      const message = parseTrpcError(err);
      toast.error(message);
    },
  });

  const updateDeletedAt = trpc.key.update.deletedAt.useMutation({
    onSuccess() {
      toast.success("Key Rerolled.");
    },
    onError(err) {
      console.error(err);
      const message = parseTrpcError(err);
      toast.error(message);
    },
  });

  const updateExpiration = trpc.key.update.expiration.useMutation({
    onSuccess() {
      toast.success("Key Rerolled.");
    },
    onError(err) {
      console.error(err);
      const message = parseTrpcError(err);
      toast.error(message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    const ratelimit = apiKey.ratelimitLimit
      ? {
          async: apiKey.ratelimitAsync ?? false,
          duration: apiKey.ratelimitDuration ?? 0,
          limit: apiKey.ratelimitLimit ?? 0,
        }
      : undefined;

    const refill = apiKey.refillInterval
      ? {
          interval: apiKey.refillInterval ?? "daily",
          amount: apiKey.refillAmount ?? 0,
        }
      : undefined;

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

    apiKey.roles?.forEach(async (role) => {
      await copyRolesToNewKey.mutateAsync({ roleId: role.id, keyId: newKey.keyId });
    });

    apiKey.permissions?.forEach(async (permission) => {
      await copyPermissionsToNewKey.mutateAsync({
        permission: permission.name,
        keyId: newKey.keyId,
      });
    });

    if (apiKey.encrypted) {
      await copyEncryptedToNewKey.mutateAsync({
        encrypted: apiKey.encrypted.encrypted,
        encryptiodKeyId: apiKey.encrypted.encryptionKeyId,
        keyId: newKey.keyId,
      });
    }

    if (values.expiresIn === "now") {
      await updateDeletedAt.mutateAsync({
        keyId: apiKey.id,
        deletedAt: new Date(Date.now()),
      });
    } else {
      const miliseconds = ms(values.expiresIn);
      const expiration = new Date(Date.now() + miliseconds);

      await updateExpiration.mutateAsync({
        keyId: apiKey.id,
        expiration,
        enableExpiration: true, 
      })
    }
  }

  const [confirmatioDialogOpen, setConfirmationDialogOpen] = useState(false);
  const confirmationSubmit = () => {
    setConfirmationDialogOpen(false);
    onSubmit(form.getValues());
  };

  return (
    <>
      <RerollConfirmationDialog
        open={confirmatioDialogOpen}
        setOpen={setConfirmationDialogOpen}
        onClick={confirmationSubmit}
        lastUsed={lastUsed}
      />
      <RerollNewKeyDialog newKey={createKey.data} apiId={apiId} keyAuthId={apiKey.keyAuthId} />
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
              <Button type="button" variant="alert" onClick={() => setConfirmationDialogOpen(true)}>
                Reroll Key
              </Button>
            </CardFooter>
          </Card>
        </form>
      </Form>
    </>
  );
};
