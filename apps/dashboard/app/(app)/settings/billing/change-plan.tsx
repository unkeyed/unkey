"use client";
import { Loading } from "@/components/dashboard/loading";
import { Dialog, DialogContent, DialogFooter, DialogTrigger } from "@/components/ui/dialog";
import { Form, FormDescription, FormField, FormItem, FormLabel } from "@/components/ui/form";
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
import { Button } from "@unkey/ui";
import Link from "next/link";
import { useRouter } from "next/navigation";
import type React from "react";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { z } from "zod";
const formSchema = z.object({
  priceId: z.string(),
});

type Props = {
  currentPriceId?: string;
  prices: Array<{ label: string; priceId: string }>;
};

export const ChangePlanButton = ({
  currentPriceId,
  prices,
  ...rest
}: React.ButtonHTMLAttributes<HTMLButtonElement> & Props) => {
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      priceId: currentPriceId,
    },
  });

  const [open, setOpen] = useState(false);

  const router = useRouter();
  const changePlan = trpc.workspace.updatePlan.useMutation({
    async onSuccess() {
      toast.success("Your plan was successfully changed");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
    onSettled() {
      setOpen(false);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    changePlan.mutate(values);
  }
  return (
    <>
      <Dialog open={open} onOpenChange={(o) => setOpen(o)}>
        <DialogTrigger asChild>
          <Button {...rest}>Change Plan</Button>
        </DialogTrigger>
        <DialogContent className="border-border w-11/12 max-sm: ">
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)}>
              <FormField
                control={form.control}
                name="priceId"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Change Plan</FormLabel>

                    <Select onValueChange={field.onChange} value={field.value}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {prices.map((p) => (
                          <SelectItem
                            key={p.priceId}
                            value={p.priceId}
                            disabled={p.priceId === currentPriceId}
                          >
                            {p.label}
                          </SelectItem>
                        ))}{" "}
                      </SelectContent>
                    </Select>
                    <FormDescription>Select a new plan</FormDescription>
                  </FormItem>
                )}
              />

              <DialogFooter className="flex w-full items-center justify-between gap-2 pt-4 ">
                <Link href="/settings/billing/stripe">
                  <Button variant="destructive">Cancel Subscription</Button>
                </Link>
                <Button
                  variant="primary"
                  disabled={changePlan.isLoading || !form.formState.isValid}
                  type="submit"
                >
                  {changePlan.isLoading ? <Loading /> : "Change Plan"}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>
    </>
  );
};
