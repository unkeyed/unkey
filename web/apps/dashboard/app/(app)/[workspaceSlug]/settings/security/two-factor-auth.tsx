"use client";

import { trpc } from "@/lib/trpc/client";
import { Button, DialogContainer, FormInput, SettingCard, toast } from "@unkey/ui";
import { useState } from "react";

export function TwoFactorAuth() {
  const utils = trpc.useUtils();
  const { data: factors, isLoading } = trpc.user.mfa.listFactors.useQuery();

  const removeFactor = trpc.user.mfa.removeFactor.useMutation({
    async onSuccess() {
      toast.success("Authenticator app removed");
      await utils.user.mfa.listFactors.refetch();
    },
    onError(err) {
      toast.error("Failed to remove authenticator app", {
        description: err.message,
      });
    },
  });

  return (
    <SettingCard
      title="Two-factor authentication"
      description="Require a one-time code from an authenticator app when signing in to your account."
      border="both"
      contentWidth="w-full lg:w-[420px]"
    >
      <div className="flex flex-col w-full gap-3">
        {isLoading ? (
          <div className="text-gray-9 text-[13px]">Loading...</div>
        ) : factors && factors.length > 0 ? (
          factors.map((factor) => (
            <div
              key={factor.id}
              className="flex items-center justify-between w-full gap-2 border border-grayA-4 rounded-lg px-3 py-2"
            >
              <div className="flex flex-col">
                <span className="text-accent-12 text-[13px] font-medium">Authenticator app</span>
                <span className="text-gray-9 text-xs">
                  Added {new Date(factor.createdAt).toLocaleDateString()}
                </span>
              </div>
              <Button
                variant="destructive"
                size="md"
                loading={removeFactor.isLoading}
                onClick={() => {
                  if (
                    window.confirm(
                      "Remove this authenticator app? You will no longer be asked for a code when signing in.",
                    )
                  ) {
                    removeFactor.mutate({ factorId: factor.id });
                  }
                }}
              >
                Remove
              </Button>
            </div>
          ))
        ) : (
          <div className="flex items-center justify-between w-full gap-2">
            <span className="text-gray-9 text-[13px]">No authenticator app configured</span>
            <EnrollDialog />
          </div>
        )}
      </div>
    </SettingCard>
  );
}

function EnrollDialog() {
  const utils = trpc.useUtils();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [code, setCode] = useState("");

  const startEnrollment = trpc.user.mfa.startEnrollment.useMutation({
    onError(err) {
      toast.error("Failed to start enrollment", { description: err.message });
      setDialogOpen(false);
    },
  });

  const verifyEnrollment = trpc.user.mfa.verifyEnrollment.useMutation({
    async onSuccess({ valid }) {
      if (!valid) {
        toast.error("Invalid code, please try again");
        return;
      }
      toast.success("Two-factor authentication enabled");
      setDialogOpen(false);
      setCode("");
      await utils.user.mfa.listFactors.refetch();
    },
    onError(err) {
      toast.error("Failed to verify the code", { description: err.message });
    },
  });

  const enrollment = startEnrollment.data;

  const openDialog = () => {
    setDialogOpen(true);
    setCode("");
    startEnrollment.reset();
    startEnrollment.mutate();
  };

  return (
    <>
      <Button variant="primary" size="md" onClick={openDialog}>
        Add authenticator app
      </Button>
      <DialogContainer
        isOpen={dialogOpen}
        onOpenChange={(open) => setDialogOpen(open)}
        title="Set up two-factor authentication"
        subTitle="Scan the QR code with your authenticator app, then enter the 6 digit code to confirm."
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              form="mfa-enroll-form"
              variant="primary"
              size="xlg"
              type="submit"
              className="w-full rounded-lg"
              disabled={!enrollment || code.length !== 6 || verifyEnrollment.isLoading}
              loading={verifyEnrollment.isLoading}
            >
              Verify and enable
            </Button>
          </div>
        }
      >
        {startEnrollment.isLoading || !enrollment ? (
          <div className="text-gray-9 text-[13px] text-center py-8">Generating QR code...</div>
        ) : (
          <form
            id="mfa-enroll-form"
            className="flex flex-col items-center gap-4"
            onSubmit={(e) => {
              e.preventDefault();
              verifyEnrollment.mutate({
                factorId: enrollment.factorId,
                challengeId: enrollment.challengeId,
                code,
              });
            }}
          >
            <img
              src={enrollment.qrCode}
              alt="QR code for authenticator app"
              className="w-44 h-44 rounded-lg bg-white p-2"
            />
            <p className="text-gray-9 text-xs break-all text-center">
              Can't scan it? Enter this secret manually:{" "}
              <span className="font-mono text-accent-12">{enrollment.secret}</span>
            </p>
            <FormInput
              label="Verification code"
              placeholder="123456"
              inputMode="numeric"
              autoComplete="one-time-code"
              maxLength={6}
              value={code}
              onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
              className="w-full"
            />
          </form>
        )}
      </DialogContainer>
    </>
  );
}
