"use client";

import { Overlay } from "@/components/consent/overlay";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useConsentManager } from "@koroflow/core-react";
import { AnimatePresence, motion } from "framer-motion";
import { X } from "lucide-react";
import * as React from "react";
import { createPortal } from "react-dom";
import ConsentCustomizationWidget from "./consent-customization-widget";
import { usePostHog } from "posthog-js/react";

export interface ConsentCustomizationDialogProps {
  children?: React.ReactNode;
  triggerClassName?: string;
  showCloseButton?: boolean;
  asChild?: boolean;
}

const dialogVariants = {
  hidden: { opacity: 0 },
  visible: { opacity: 1 },
  exit: { opacity: 0 },
};

const contentVariants = {
  hidden: { opacity: 0, scale: 0.95 },
  visible: {
    opacity: 1,
    scale: 1,
    transition: { type: "spring", stiffness: 300, damping: 30 },
  },
  exit: {
    opacity: 0,
    scale: 0.95,
    transition: { duration: 0.2 },
  },
};

const ConsentCustomizationCard = ({
  onClose,
  showCloseButton,
  handleSave,
  ref,
}: {
  onClose: () => void;
  showCloseButton: boolean;
  handleSave: () => void;
  ref: React.RefObject<HTMLDivElement>;
}) => {
  const onSaveWrapper = React.useCallback(() => {
    handleSave();
  }, [handleSave]);

  const onCloseWrapper = React.useCallback(() => {
    onClose();
  }, [onClose]);

  return (
    <Card
      className="w-full relative border bg-gradient-to-b from-[#111111] to-black border-t-[.75px] border-white/20 overflow-hidden"
      ref={ref}
    >
      <CardHeader className="space-y-2 pb-0 border-white/10 editor-top-gradient">
        {showCloseButton && (
          <Button
            variant="ghost"
            size="icon"
            className="absolute right-2 top-2"
            onClick={onCloseWrapper}
            aria-label="Close privacy settings"
          >
            <X className="h-4 w-4" />
          </Button>
        )}
        <CardTitle id="privacy-settings-title">Privacy Settings</CardTitle>
        <CardDescription>
          Customize your privacy settings here. You can choose which types of
          cookies and tracking technologies you allow.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <ConsentCustomizationWidget onSave={onSaveWrapper} />
      </CardContent>
    </Card>
  );
};

export const ConsentCustomizationDialog = React.forwardRef<
  HTMLDivElement,
  ConsentCustomizationDialogProps
>(({ showCloseButton = false }, ref) => {
  const posthog = usePostHog();

  const { isPrivacyDialogOpen, setIsPrivacyDialogOpen, saveConsents } =
    useConsentManager();
  const [isMounted, setIsMounted] = React.useState(false);
  const contentRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    setIsMounted(true);
    return () => setIsMounted(false);
  }, []);

  const handleSave = React.useCallback(() => {
    saveConsents("custom");
    posthog.capture("consent", {
      consent: "custom",
    });
    setIsPrivacyDialogOpen(false);
  }, [setIsPrivacyDialogOpen, saveConsents, posthog]);

  const handleClose = React.useCallback(() => {
    setIsPrivacyDialogOpen(false);
  }, [setIsPrivacyDialogOpen]);

  const dialogContent = (
    <AnimatePresence mode="wait">
      {isPrivacyDialogOpen && (
        <>
          <Overlay show={isPrivacyDialogOpen} />
          <motion.dialog
            className="fixed inset-0 z-50 flex items-center justify-center"
            variants={dialogVariants}
            initial="hidden"
            animate="visible"
            exit="exit"
            aria-modal="true"
            aria-labelledby="privacy-settings-title"
            onClick={(e) => {
              // Close dialog when clicking outside
              if (e.target === e.currentTarget) {
                handleClose();
              }
            }}
          >
            <motion.div
              ref={contentRef}
              className="z-50 w-full max-w-md mx-auto"
              variants={contentVariants}
              initial="hidden"
              animate="visible"
              exit="exit"
            >
              <ConsentCustomizationCard
                ref={ref as React.RefObject<HTMLDivElement>}
                onClose={handleClose}
                showCloseButton={showCloseButton}
                handleSave={handleSave}
              />
            </motion.div>
          </motion.dialog>
        </>
      )}
    </AnimatePresence>
  );

  return isMounted && createPortal(dialogContent, document.body);
});

ConsentCustomizationDialog.displayName = "ConsentCustomizationDialog";
