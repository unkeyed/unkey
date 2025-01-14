"use client";

// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";

type ActionProps = {
  className?: string;
  children?: React.ReactNode;
  onApply: () => void;
};

const Actions: React.FC<ActionProps> = ({ className, children, onApply }) => {
  return <div>Actions</div>;
}
export { Actions };
