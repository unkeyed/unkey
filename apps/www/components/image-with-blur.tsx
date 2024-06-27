import Image, { type ImageProps } from "next/image";
import React from "react";

type ImageWithBlurProps = ImageProps;

export function ImageWithBlur(props: ImageWithBlurProps) {
  return (
    <Image
      {...props}
      placeholder="blur"
      blurDataURL="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8+e1bKQAJMQNc5W2CQwAAAABJRU5ErkJggg=="
    />
  );
}
