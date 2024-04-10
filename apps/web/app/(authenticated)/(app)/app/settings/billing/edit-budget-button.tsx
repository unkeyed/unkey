"use client";

import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";

import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
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
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import type { Budget } from "@unkey/db";
import { Edit } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { createBudgetFormSchema } from "./create-budget-button";

const editBudgetFormSchema = createBudgetFormSchema.extend({
  enabled: z.boolean(),
});

type Props = {
  budget: Budget;
};

export const EditBudgetButton: React.FC<Props> = ({ budget }) => {
  const router = useRouter();

  const [isOpen, setIsOpen] = useState(false);

  const form = useForm<z.infer<typeof editBudgetFormSchema>>({
    resolver: zodResolver(editBudgetFormSchema),
    defaultValues: {
      name: budget.name || "",
      enabled: budget.enabled,
      fixedAmount: budget.fixedAmount,
      additionalEmails: budget.data.additionalEmails?.join(",") || "",
    },
  });

  const editBudget = trpc.budget.update.useMutation({
    onSuccess() {
      toast.success("Your Budget has been updated");

      router.refresh();

      setIsOpen(false);
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof editBudgetFormSchema>) {
    await editBudget.mutateAsync({
      ...values,
      budgetId: budget.id,
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
      <Button size="icon" type="button" onClick={() => setIsOpen(true)} variant="secondary">
        <Edit className="w-4 h-4" />
      </Button>

      <DialogContent className="w-11/12 max-sm: ">
        <DialogHeader>
          <DialogTitle>Edit Budget</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="enabled"
              render={({ field }) => (
                <FormItem className="w-full">
                  <div className="flex items-center gap-4">
                    <FormControl>
                      <Switch
                        id="enabled"
                        checked={form.getValues("enabled")}
                        onCheckedChange={(e) => {
                          field.onChange(e);
                        }}
                      />
                    </FormControl>{" "}
                    <FormLabel htmlFor="enabled">
                      {form.getValues("enabled") ? "Enabled" : "Disabled"}
                    </FormLabel>
                  </div>
                </FormItem>
              )}
            />

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
                disabled={editBudget.isLoading || form.formState.isSubmitting}
                className="mt-4 "
                type="submit"
              >
                {editBudget.isLoading ? <Loading /> : "Save"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};
