"use client";;
import { Loading } from "@/components/dashboard/loading";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { FormField } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { useTRPC } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useMutation } from "@tanstack/react-query";
const formSchema = z.object({
  keyAuthId: z.string(),
  defaultBytes: z
    .number()
    .min(16, "Byte size needs to be at least 16")
    .max(255, "Byte size cannot exceed 255")
    .optional(),
});

type Props = {
  keyAuth: {
    id: string;
    defaultBytes: number | undefined | null;
  };
};

export const DefaultBytes: React.FC<Props> = ({ keyAuth }) => {
  const trpc = useTRPC();
  const router = useRouter();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      defaultBytes: keyAuth.defaultBytes ?? undefined,
      keyAuthId: keyAuth.id,
    },
  });

  const setDefaultBytes = useMutation(trpc.api.setDefaultBytes.mutationOptions({
    onSuccess() {
      toast.success("Default Byte length for this API is updated!");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  }));

  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (values.defaultBytes === keyAuth.defaultBytes || !values.defaultBytes) {
      return toast.error(
        "Please provide a different byte-size than already existing one as default",
      );
    }
    await setDefaultBytes.mutateAsync(values);
  }

  return (
    (<form onSubmit={form.handleSubmit(onSubmit)}>
      <Card>
        <CardHeader>
          <CardTitle>Default Bytes</CardTitle>
          <CardDescription>
            Set default Bytes for the keys under this API. Default byte size must be between{" "}
            <span className="font-bold">8 to 255</span>
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col space-y-2">
            <input type="hidden" name="keyAuthId" value={keyAuth.id} />
            <label className="hidden sr-only">Default Bytes</label>
            <FormField
              control={form.control}
              name="defaultBytes"
              render={({ field }) => (
                <Input
                  className="max-w-sm"
                  {...field}
                  autoComplete="off"
                  onChange={(e) => field.onChange(Number(e.target.value.replace(/\D/g, "")))}
                />
              )}
            />
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <Button
            variant="primary"
            disabled={!form.formState.isValid || form.formState.isSubmitting}
            type="submit"
          >
            {form.formState.isSubmitting ? <Loading /> : "Save"}
          </Button>
        </CardFooter>
      </Card>
    </form>)
  );
};
