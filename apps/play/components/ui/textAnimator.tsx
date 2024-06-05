import { TypeAnimation } from "react-type-animation";
type TextAnimatorProps = {
  input: string;
  repeat: number;
  style: string;
};
export default function TextAnimator({ input, repeat = 0, style }: TextAnimatorProps) {
  return (
    <TypeAnimation
      className={style}
      sequence={[input]}
      cursor={false}
      speed={99}
      style={{
        whiteSpace: "pre",
        fontSize: "1em",
        fontFamily: "GeistMono, monospace",
      }}
      repeat={repeat}
    />
  );
}
