import Image, { type ImageProps } from "next/image";

export const ImageWithBlur: React.FC<ImageProps> = (props) => {
  return (
    <Image
      {...props}
      placeholder="blur"
      blurDataURL="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8+e1bKQAJMQNc5W2CQwAAAABJRU5ErkJggg=="
    />
  );
};
