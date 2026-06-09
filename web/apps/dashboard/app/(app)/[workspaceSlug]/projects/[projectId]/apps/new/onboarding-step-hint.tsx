import type { ReactNode } from "react";

type OnboardingStepHintProps = {
  children: ReactNode;
};

export const OnboardingStepHint = ({ children }: OnboardingStepHintProps) => (
  <div className="mt-8 flex justify-center">
    <span className="text-sm text-gray-11">{children}</span>
  </div>
);

export const OnboardingStepHintHighlight = ({ children }: OnboardingStepHintProps) => (
  <span className="font-medium text-gray-12 underline underline-offset-2 decoration-grayA-6 group-hover:decoration-gray-12 transition-colors decoration-dotted">
    {children}
  </span>
);
