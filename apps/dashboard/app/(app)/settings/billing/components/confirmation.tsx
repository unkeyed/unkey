"use client";

import { Button, DialogContainer } from "@unkey/ui";
import { useState } from "react";

type Props = {
  title: string;

  description: string;
  fineprint?: string;

  onConfirm: () => Promise<void>;

  trigger: (onClick: () => void) => React.ReactNode;
};

export const Confirm: React.FC<Props> = (props) => {
  const [isOpen, setOpen] = useState(false);
  const [isLoading, setLoading] = useState(false);

  return (
    <>
      {props.trigger(() => setOpen(true))}
      <DialogContainer
        isOpen={isOpen}
        onOpenChange={setOpen}
        title={props.title}
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              type="button"
              variant="primary"
              size="xlg"
              loading={isLoading}
              className="w-full rounded-lg"
              onClick={async () => {
                setLoading(true);
                setOpen(false);
                await props.onConfirm();
                setLoading(false);
              }}
            >
              Confirm
            </Button>
            <div className="text-gray-9 text-xs">{props.fineprint}</div>
          </div>
        }
      >
        <p className="text-gray-11 text-[13px]">{props.description}</p>
      </DialogContainer>
    </>
  );
};
