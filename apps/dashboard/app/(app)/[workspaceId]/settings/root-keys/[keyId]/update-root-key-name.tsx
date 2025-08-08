"use client";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  Button,
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
  FormInput,
  toast,
} from "@unkey/ui";
import { useRouter } from "next/navigation";
import { Controller, useForm } from "react-hook-form";
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

  const {
    handleSubmit,
    control,
    formState: { errors, isValid },
  } = useForm<z.infer<typeof formSchema>>({
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
    <form onSubmit={handleSubmit(onSubmit)}>
      <Card>
        <CardHeader>
          <CardTitle>Name</CardTitle>
        </CardHeader>
        <CardContent className="flex justify-between item-center pt-2 mb-10">
          <div className={cn("flex flex-col space-y-2 w-full")}>
            <input type="hidden" name="keyId" value={apiKey.id} />

            <Controller
              control={control}
              name="name"
              render={({ field }) => (
                <FormInput
                  {...field}
                  type="string"
                  className="h-8 max-w-sm"
                  autoComplete="off"
                  description="Give your root key a name. This is optional and not customer facing."
                  wrapperClassName="w-full h-full"
                  error={errors.name?.message}
                />
              )}
            />
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <Button
            disabled={updateName.isLoading || !isValid}
            type="submit"
            loading={updateName.isLoading}
          >
            Save
          </Button>
        </CardFooter>
      </Card>
    </form>
  );
};
