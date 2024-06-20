import { TypeAnimation } from "react-type-animation";
type TextAnimatorProps = {
  input: string;
  repeat: number;
  style: string;
};
export default function TextAnimator({ input, repeat = 0, style }: TextAnimatorProps) {
  return (
    <TypeAnimation
      className={style.toString()}
      sequence={[input]}
      cursor={false}
      speed={99}
      repeat={repeat}
    />
  );
}
