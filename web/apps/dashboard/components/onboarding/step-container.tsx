import type { ReactNode } from "react";

type OnboardingStepContainerProps = {
  children: ReactNode;
};

export const OnboardingStepContainer = ({ children }: OnboardingStepContainerProps) => (
  <div className="flex flex-col items-center justify-center my-40 gap-6">{children}</div>
);
