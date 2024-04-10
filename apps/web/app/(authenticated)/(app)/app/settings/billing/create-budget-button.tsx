"use client";

import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";

import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
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
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

export const createBudgetFormSchema = z.object({
  name: z.string().optional(),
  fixedAmount: z.coerce
    .number({
      errorMap: (issue, { defaultError }) => ({
        message:
          issue.code === "invalid_type" ? "Budget amount must be greater than 0" : defaultError,
      }),
    })
    .positive({ message: "Budget amount must be greater than 0" }),
  additionalEmails: z
    .string()
    .optional()
    .refine(
      (val) =>
        !val ||
        z.array(z.string().email("Invalid email format")).max(10).safeParse(val.split(",")).success,
      "Invalid email list. Ensure emails are correctly formatted and do not exceed 10.",
    ),
});

export const CreateBudgetButton: React.FC<React.ButtonHTMLAttributes<HTMLButtonElement>> = (
  props,
) => {
  const router = useRouter();

  const [isOpen, setIsOpen] = useState(false);

  const form = useForm<z.infer<typeof createBudgetFormSchema>>({
    resolver: zodResolver(createBudgetFormSchema),
    defaultValues: {
      additionalEmails: "",
    },
  });

  const createBudget = trpc.budget.create.useMutation({
    onSuccess() {
      toast.success("Your Budget has been created");

      router.refresh();

      setIsOpen(false);
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof createBudgetFormSchema>) {
    await createBudget.mutateAsync({
      ...values,
      additionalEmails: values.additionalEmails?.split(",").filter(Boolean) || undefined,
    });
  }

  function onOpenChange(open: boolean) {
    if (open) {
      form.reset({
        fixedAmount: undefined,
        additionalEmails: "",
      });
    }
    setIsOpen(open);
  }

  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogTrigger asChild>
        <Button className="flex-row items-center gap-1 font-semibold " {...props}>
          Create Budget
        </Button>
      </DialogTrigger>

      <DialogContent className="w-11/12 max-sm: ">
        <DialogHeader>
          <DialogTitle>Create a new Budget</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Budget Name</FormLabel>
                  <FormControl>
                    <Input {...field} autoComplete="off" />
                  </FormControl>
                  <FormDescription>Provide a descriptive name for this budget.</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="fixedAmount"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Budget amount ($)</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      min={0}
                      className="max-w-[120px]"
                      {...field}
                      autoComplete="off"
                      placeholder="0.00"
                      step="0.01"
                    />
                  </FormControl>
                  <FormDescription>Enter your budgeted amount.</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="additionalEmails"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Email recipients</FormLabel>
                  <FormControl>
                    <Textarea
                      rows={3}
                      {...field}
                      autoComplete="off"
                      placeholder="Separate email addresses using commas"
                    />
                  </FormControl>
                  <FormDescription>
                    Specify up to 10 additional email recipients you want to notify when the
                    threshold has exceeded. The account owner will also get notified.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <DialogFooter className="flex-row justify-end gap-2 pt-4 ">
              <Button
                disabled={createBudget.isLoading || form.formState.isSubmitting}
                className="mt-4 "
                type="submit"
              >
                {createBudget.isLoading ? <Loading /> : "Create"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};
