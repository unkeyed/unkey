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
  name: z
    .string()
    .transform((e) => (e === "" ? undefined : e))
    .optional(),
});
type Props = {
  apiKey: {
    id: string;
    name: string | null;
  };
};

export const UpdateRootKeyName: React.FC<Props> = ({ apiKey }) => {
  const router = useRouter();

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    defaultValues: {
      keyId: apiKey.id,
      name: apiKey.name ?? "",
    },
  });

  const updateName = trpc.rootKey.update.name.useMutation({
    onSuccess() {
      toast.success("Your root key name has been updated!");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    await updateName.mutateAsync(values);
  }
  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <Card>
          <CardHeader>
            <CardTitle>Name</CardTitle>
            <CardDescription>
              Give your root key a name. This is optional and not customer facing.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex justify-between item-center">
            <div className={cn("flex flex-col space-y-2 w-full ")}>
              <input type="hidden" name="keyId" value={apiKey.id} />
              <Label htmlFor="remaining">Name</Label>
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <Input {...field} type="string" className="h-8 max-w-sm" autoComplete="off" />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </CardContent>
          <CardFooter className="justify-end">
            <Button disabled={updateName.isLoading || !form.formState.isValid} type="submit">
              {updateName.isLoading ? <Loading /> : "Save"}
            </Button>
          </CardFooter>
        </Card>
      </form>
    </Form>
  );
};
