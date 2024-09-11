import Image, { type ImageProps } from "next/image";

export function ImageWithBlur(props: ImageProps){
  return props.fill ? (<Image
    {...props}
   fill={true}
    className="rounded-md"
    placeholder="blur"
    blurDataURL="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8+e1bKQAJMQNc5W2CQwAAAABJRU5ErkJggg=="
  />) :
(
  <Image
  {...props}
  width={1920}
  height={1080}
  className="rounded-md"
  placeholder="blur"
  blurDataURL="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8+e1bKQAJMQNc5W2CQwAAAABJRU5ErkJggg=="
/>
  );
};

