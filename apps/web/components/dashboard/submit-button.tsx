"use client";
// @ts-expect-error
import { experimental_useFormStatus } from "react-dom";
import { Button, type ButtonProps } from "../ui/button";
import { Loading } from "./loading";
type Props = { label: string; variant?: ButtonProps["variant"] };

export const SubmitButton: React.FC<Props> = ({ label, variant }) => {
  const { pending } = experimental_useFormStatus();
  return (
    <Button variant={pending ? "disabled" : variant ?? "primary"} type="submit" disabled={pending}>
      {pending ? <Loading /> : label}
    </Button>
  );
};
