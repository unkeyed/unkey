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

export const ImageWithZoom: React.FC<ImageProps & { zoomScale?: number }> = ({
  zoomScale = 1.5,
  className = "",
  ...props
}) => {
  return (
    <div className="overflow-hidden">
      <div
        className={`transition-transform duration-300 ease-in-out hover:scale-${zoomScale} ${className}`}
      >
        <ImageWithBlur {...props} />
      </div>
    </div>
  );
};