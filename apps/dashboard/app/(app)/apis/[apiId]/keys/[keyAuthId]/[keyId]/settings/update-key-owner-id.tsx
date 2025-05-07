"use client";
import { Loading } from "@/components/dashboard/loading";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Form, FormControl, FormField, FormItem, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  keyId: z.string(),
  ownerId: z
    .string()
    .trim()
    .transform((e) => (e === "" ? undefined : e))
    .optional(),
});

type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    ownerId: string | null;
  };
};

export const UpdateKeyOwnerId: React.FC<Props> = ({ apiKey }) => {
  const router = useRouter();

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    defaultValues: {
      keyId: apiKey.id,
      ownerId: apiKey.ownerId ?? "",
    },
  });

  const updateOwnerId = trpc.key.update.ownerId.useMutation({
    onSuccess() {
      toast.success("Your owner ID has been updated!");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    await updateOwnerId.mutateAsync({ ...values, ownerType: "v1" });
  }
  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <Card>
          <CardHeader>
            <CardTitle>Owner ID</CardTitle>
            <CardDescription>
              Use this to identify the owner of the key. For example by setting the userId of the
              user in your system.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex justify-between item-center">
            <div className={cn("flex flex-col space-y-2 w-full ")}>
              <input type="hidden" name="keyId" value={apiKey.id} />
              <Label htmlFor="ownerId">OwnerId</Label>
              <FormField
                control={form.control}
                name="ownerId"
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <Input
                        {...field}
                        type="string"
                        className="h-8 max-w-sm"
                        defaultValue={apiKey.ownerId ?? ""}
                        autoComplete="off"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </CardContent>
          <CardFooter className="justify-end">
            <Button disabled={updateOwnerId.isLoading || !form.formState.isValid} type="submit">
              {updateOwnerId.isLoading ? <Loading /> : "Save"}
            </Button>
          </CardFooter>
        </Card>
      </form>
    </Form>
  );
};
