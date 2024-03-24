"use client";
import { Button } from "@/components/ui/button";
import React, { useState } from "react";

import { toast } from "@/components/ui/toaster";

import { Loading } from "@/components/dashboard/loading";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { trpc } from "@/lib/trpc/client";
import { Trash } from "lucide-react";
import { useRouter } from "next/navigation";

type Props = {
  budgetId: string;
};

export const DeleteBudgetButton: React.FC<Props> = ({ budgetId }) => {
  const router = useRouter();
  const [isOpen, setIsOpen] = useState(false);

  const deleteBudget = trpc.budget.delete.useMutation({
    onSuccess() {
      toast.success("Budget deleted");

      router.refresh();

      setIsOpen(false);
    },
    onError(error) {
      console.error(error);
      toast.error(error.message);
    },
  });

  return (
    <>
      <Button size="icon" type="button" onClick={() => setIsOpen(true)} variant="alert">
        <Trash className="w-4 h-4" />
      </Button>

      <Dialog open={isOpen} onOpenChange={setIsOpen}>
        <DialogContent className="border-[#b80f07]">
          <DialogHeader>
            <DialogTitle>Delete Budget</DialogTitle>
            <DialogDescription>
              This budget will be deleted. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>

          <DialogFooter className="justify-end">
            <Button type="button" onClick={() => setIsOpen(false)} variant="secondary">
              Cancel
            </Button>
            <Button
              type="submit"
              variant="alert"
              disabled={deleteBudget.isLoading}
              onClick={() => deleteBudget.mutate({ budgetId })}
            >
              {deleteBudget.isLoading ? <Loading /> : "Delete Budget"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
};
