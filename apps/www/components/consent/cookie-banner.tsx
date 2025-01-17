"use client";

import { useConsentManager } from "@koroflow/core-react";
import { AnimatePresence, motion } from "framer-motion";
import { X } from "lucide-react";
import * as React from "react";
import { createPortal } from "react-dom";

import { Button } from "@/components/ui/button";
import { Card, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { PrimaryButton, SecondaryButton } from "../button";

type HorizontalPosition = "left" | "center" | "right";
type VerticalPosition = "top" | "bottom";

interface PrivacyPopupProps extends React.HTMLAttributes<HTMLDivElement> {
  bannerDescription?: string;
  bannerTitle?: string;
  horizontalPosition?: HorizontalPosition;
  verticalPosition?: VerticalPosition;
  showCloseButton?: boolean;
}

const CookieBanner = React.forwardRef<HTMLDivElement, PrivacyPopupProps>(
  (
    {
      bannerDescription = "This site uses cookies and similar technologies to measure and improve your experience.",
      bannerTitle = "We value your privacy",
      className,
      horizontalPosition = "left",
      verticalPosition = "bottom",
      showCloseButton = false,
    },
    ref,
  ) => {
    const {
      showPopup,
      setIsPrivacyDialogOpen,
      setShowPopup,
      saveConsents,
      setConsent,
      callbacks,
      complianceSettings,
      isPrivacyDialogOpen,
      hasConsented,
      consents,
    } = useConsentManager();

    const bannerShownRef = React.useRef(false);
    const [isMounted, setIsMounted] = React.useState(false);

    React.useEffect(() => {
      setIsMounted(true);
      return () => setIsMounted(false);
    }, []);

    React.useEffect(() => {
      if (!isMounted) {
        return;
      }

      if (showPopup && !bannerShownRef.current && !hasConsented()) {
        callbacks.onBannerShown?.();
        bannerShownRef.current = true;
      }
    }, [showPopup, callbacks, hasConsented, isMounted]);

    const acceptAll = React.useCallback(() => {
      const allConsents = Object.keys(consents) as (keyof typeof consents)[];
      for (const consentName of allConsents) {
        setConsent(consentName, true);
      }
      saveConsents("all");
    }, [consents, setConsent, saveConsents]);

    const rejectAll = React.useCallback(() => {
      saveConsents("necessary");
    }, [saveConsents]);

    const handleClose = React.useCallback(() => {
      setShowPopup(false);
      callbacks.onBannerClosed?.();
    }, [setShowPopup, callbacks]);

    const handleCustomize = React.useCallback(() => {
      setIsPrivacyDialogOpen(true);
    }, [setIsPrivacyDialogOpen]);

    const positionClasses = cn(
      "fixed z-50 max-w-md",
      {
        "left-4": horizontalPosition === "left",
        "right-4": horizontalPosition === "right",
        "left-1/2 -translate-x-1/2": horizontalPosition === "center",
        "top-4": verticalPosition === "top",
        "bottom-4": verticalPosition === "bottom",
      },
      className,
    );

    // Early return for SSR and when user has consented
    if (!isMounted || (hasConsented() && !showPopup)) {
      return null;
    }

    const BannerContent = () => (
      <AnimatePresence>
        {showPopup && !isPrivacyDialogOpen && (
          <motion.dialog
            className="fixed inset-0 z-50 flex items-end sm:items-center justify-center px-4 sm:px-0"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            aria-modal="true"
            aria-labelledby="cookie-consent-title"
          >
            <motion.div
              className={positionClasses}
              initial={{ opacity: 0, y: 50 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: 50 }}
              transition={{ type: "spring", stiffness: 300, damping: 30 }}
              ref={ref}
            >
              <Card className="w-full relative border bg-gradient-to-b from-[#111111] to-black border-t-[.75px] border-white/20 overflow-hidden">
                <CardHeader className="space-y-2 pb-0 border-white/10 editor-top-gradient">
                  {showCloseButton && (
                    <Button
                      variant="ghost"
                      size="icon"
                      className="absolute right-2 top-2"
                      onClick={handleClose}
                      aria-label="Close cookie consent banner"
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  )}
                  <CardTitle id="cookie-consent-title" className="text-lg sm:text-xl">
                    {bannerTitle}
                  </CardTitle>
                  <CardDescription className="text-sm sm:text-base">
                    {bannerDescription}
                  </CardDescription>
                </CardHeader>
                <CardFooter className="flex flex-col sm:flex-row justify-between gap-4 p-4 sm:p-6">
                  <div className="flex flex-col sm:flex-row justify-between gap-2 w-full sm:w-auto">
                    {complianceSettings.gdpr.enabled && (
                      <SecondaryButton
                        onClick={rejectAll}
                        label="Reject All"
                        className="w-full cursor-pointer text-sm sm:w-auto"
                      />
                    )}
                    <SecondaryButton
                      onClick={handleCustomize}
                      label="Customize Consent"
                      className="w-full cursor-pointer text-sm sm:w-auto"
                    />
                  </div>
                  <PrimaryButton
                    onClick={acceptAll}
                    label="Accept All"
                    className="w-full text-sm sm:w-auto cursor-pointer"
                  />
                </CardFooter>
              </Card>
            </motion.div>
          </motion.dialog>
        )}
      </AnimatePresence>
    );

    return isMounted && createPortal(<BannerContent />, document.body);
  },
);

CookieBanner.displayName = "CookieBanner";

export default CookieBanner;
