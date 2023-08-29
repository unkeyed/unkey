"use client";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import React, { useEffect, useState } from "react";
import { Loading } from "./loading";

export type ConfirmProps = {
  title: string;
  description?: string;
  trigger: React.ReactNode;
  onConfirm: () => void | Promise<void>;
  variant?: "alert";
};

export const Confirm: React.FC<ConfirmProps> = (props): JSX.Element => {
  const [isOpen, setIsOpen] = useState(false);
  const [wantOpen, _setWantOpen] = useState(isOpen);

  const [loading, setLoading] = useState(false);

  const onConfirm = async () => {
    setLoading(true);
    await props.onConfirm();
    setLoading(false);
  };

  useEffect(() => {
    if (!loading) {
      setIsOpen(wantOpen);
    }
  }, [wantOpen, loading]);

  return (
    <Dialog>
      <DialogTrigger asChild>{props.trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>{props.title}</DialogTitle>
          <DialogDescription>{props.description}</DialogDescription>
        </DialogHeader>

        <DialogFooter>
          <Button type="submit" variant={props.variant} onClick={onConfirm}>
            {loading ? <Loading /> : "Confirm"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default Confirm;
