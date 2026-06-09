import type { ReactNode } from "react";

type StepContainerProps = {
  children: ReactNode;
};

export const OnboardingStepContainer = ({ children }: StepContainerProps) => (
  <div className="flex flex-col items-center justify-center my-40 gap-6">{children}</div>
);
