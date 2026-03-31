import type { ReactNode } from "react";

type StepContainerProps = {
  children: ReactNode;
};

export const OnboardingStepContainer = ({ children }: StepContainerProps) => (
  <div className="flex flex-col items-center justify-center mt-40 my-20 gap-6">{children}</div>
);
