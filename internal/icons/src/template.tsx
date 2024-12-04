import type React from "react";
import type { IconProps } from "./props";

export const Icon: React.FC<IconProps> = (props) => {
  return <svg className={props.className} />;
};
