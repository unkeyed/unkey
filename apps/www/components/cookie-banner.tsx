"use client";

import {CookieBanner, ConsentManagerDialog} from "@c15t/react";

const CookieBannerComponent = () => {
  const secondaryButton =
    "items-center gap-2 px-4 duration-500 border border-white/40 rounded-lg text-white/70 hover:text-white h-10 flex";
  const primaryButton =
    "relative flex items-center px-4 gap-2 text-sm font-semibold text-black group-hover:bg-white/90 duration-1000 rounded-lg bg-gradient-to-r from-white/80 to-white h-10";
  return (
    <>
      <CookieBanner
        // noStyle
        // theme={{
        //   "cookie-banner.overlay": "bg-black/50 inset-0 fixed z-40",
        //   "cookie-banner.root":
        //     "bottom-12 left-12 fixed z-50 flex items-end sm:items-center justify-center px-4 sm:px-0",
        //   "cookie-banner.card":
        //     "max-w-md w-full relative border bg-gradient-to-b from-[#111111] rounded-lg to-black border-t-[.75px] border-white/20 overflow-hidden",
        //   "cookie-banner.header.root":
        //     "space-y-2 px-4 sm:px-6 pt-4 sm:pt-6 border-white/10 editor-top-gradient",
        //   "cookie-banner.header.title": "text-lg sm:text-xl",
        //   "cookie-banner.header.description": "text-sm sm:text-base",
        //   "cookie-banner.footer": "flex flex-col sm:flex-row justify-between gap-4 p-4 sm:p-6",
        //   "cookie-banner.footer.sub-group":
        //     "flex flex-col sm:flex-row justify-between gap-4 w-full sm:w-auto",
        //   "cookie-banner.footer.accept-button": primaryButton,
        //   "cookie-banner.footer.reject-button": secondaryButton,
        //   "cookie-banner.footer.customize-button": secondaryButton,
        // }}
      />
      <ConsentManagerDialog
        theme={{
          "consent-manager-dialog.root": {
            style: {
              "--c15t-card-bg-color": "black",
              "--c15t-cmw-stroke-soft-200": "rgba(255, 255, 255, 0.2)",
              "--c15t-card-border-color": "rgba(255, 255, 255, 0.2)",
            },
          },
          "consent-manager-widget.accordion": {
            style: {
              "--c15t-accordion-bg-default": "black",
              "--c15t-accordion-bg-hover": "#111111",
            },
          },
          "consent-manager-widget.footer.accept-button": {
            style: {
              "--button-bg-white": "rgba(255, 255, 255, 0.2)",
            },
          },
        }}
        //   // "consent-manager.overlay": "fixed inset-0 bg-black/50 z-[999999997]",
        //   // "consent-manager-widget.dialog":
        //   //   "fixed inset-0 z-[999999999] flex items-center justify-center ",
        //   // "consent-manager-widget.dialog.root":
        //   //   "border bg-gradient-to-b from-[#111111] rounded-lg to-black border-t-[.75px] border-white/20  rounded-lg border shadow-lg w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto",
        //   // "consent-manager-widget.dialog.header":
        //   //   "flex flex-col gap-1.5 p-6 border-b border-white/20",
        //   // "consent-manager-widget.dialog.title": "text-lg sm:text-xl",
        //   // "consent-manager-widget.dialog.description": "text-sm sm:text-base",
        //   // "consent-manager-widget.dialog.content": "px-6 py-4",
        //   // "consent-manager-widget.dialog.footer":
        //     // "flex items-center p-6 border-t border-white/20 justify-center text-center",
        //   "consent-manager-dialog.overlay": "fixed inset-0 bg-black/50 z-[999999997]",
        //   "consent-manager-dialog.root":
        //     // "fixed inset-0 z-[999999999] flex items-center justify-center ",
        //   // "consent-manager-dialog":
        //     "border bg-gradient-to-b from-[#111111] rounded-lg to-black border-t-[.75px] border-white/20  rounded-lg border shadow-lg w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto",
        //   "consent-manager-dialog.header":
        //     "flex flex-col gap-1.5 p-6 border-b border-white/20",
        //   "consent-manager-dialog.title": "text-lg sm:text-xl",
        //   "consent-manager-dialog.description": "text-sm sm:text-base",
        //   "consent-manager-dialog.content": "px-6 py-4",
        //   "consent-manager-dialog.footer":
        //     "flex items-center p-6 border-t border-white/20 justify-center text-center",
        //   "consent-manager-widget.accordion": "space-y-4",
        //   "consent-manager-widget.accordion.item": "border border-white/20 rounded-lg",
        //   // "consent-manager-widget.accordion.trigger-inner":
        //     // "w-full flex items-center justify-between hover:bg-muted/50 transition-colors",
        //   "consent-manager-widget.accordion.trigger": "flex items-center gap-4",
        //   "consent-manager-widget.accordion.icon": "w-6 h-6",
        //   "consent-manager-widget.accordion.arrow.open":
        //     "rotate-180 transition-transform duration-200",
        //   "consent-manager-widget.accordion.arrow.close": "transition-transform duration-200",
        //   "consent-manager-widget.accordion.content":
        //     "overflow-hidden text-sm transition-all data-[state=closed]:animate-accordion-up data-[state=open]:animate-accordion-down",
        //   "consent-manager-widget.accordion.content-inner": "p-4 pt-0",
        //   "consent-manager-widget.switch.track":
        //     "peer inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-[#3CEEAE]/70 data-[state=unchecked]:bg-white/20",
        //   "consent-manager-widget.switch.thumb":
        //     "pointer-events-none block h-5 w-5 rounded-full bg-white shadow-lg ring-0 transition-transform data-[state=checked]:translate-x-5 data-[state=unchecked]:translate-x-0",
        //   "consent-manager-widget.footer":
        //     "mt-6 space-y-4 flex flex-row justify-between sm:space-y-0",
        //   "consent-manager-widget.footer.sub-group": "flex flex-row justify-between gap-4",
        //   "consent-manager-widget.footer.reject-button": secondaryButton,
        //   "consent-manager-widget.footer.accept-button": secondaryButton,
        //   "consent-manager-widget.footer.customize-button": secondaryButton,
        //   "consent-manager-widget.footer.save-button": primaryButton,
        //   "consent-manager-widget.branding": "consent-manager-widget-branding",
        // }}
        // noStyle
      />
    </>
  );
};

export default CookieBannerComponent;
