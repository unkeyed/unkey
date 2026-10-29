import * as React from "react";

const PALETTE = ["#92A1C6", "#146A7C", "#F0AB3D", "#C271B4", "#C20D90"];
const BEAM_SIZE = 36;
const MARBLE_SIZE = 80;

type GeneratedAvatarProps = {
  name: string;
  variant: "beam" | "marble";
} & Omit<React.SVGProps<SVGSVGElement>, "name">;

export function GeneratedAvatar({ name, variant, ...props }: GeneratedAvatarProps) {
  const id = React.useId();
  const hash = hashName(name);
  return variant === "beam" ? (
    <BeamAvatar hash={hash} id={id} {...props} />
  ) : (
    <MarbleAvatar hash={hash} id={id} {...props} />
  );
}

type ShapeProps = { hash: number; id: string } & React.SVGProps<SVGSVGElement>;

function BeamAvatar({ hash, id, ...props }: ShapeProps) {
  const half = BEAM_SIZE / 2;
  const wrapperColor = pickColor(hash);
  const faceColor = contrastColor(wrapperColor);
  const offsetX = signedModulus(hash, 10, 1);
  const wrapperTranslateX = offsetX < 5 ? offsetX + BEAM_SIZE / 9 : offsetX;
  const offsetY = signedModulus(hash, 10, 2);
  const wrapperTranslateY = offsetY < 5 ? offsetY + BEAM_SIZE / 9 : offsetY;
  const faceTranslateX =
    wrapperTranslateX > BEAM_SIZE / 6 ? wrapperTranslateX / 2 : signedModulus(hash, 8, 1);
  const faceTranslateY =
    wrapperTranslateY > BEAM_SIZE / 6 ? wrapperTranslateY / 2 : signedModulus(hash, 7, 2);
  const mouthY = 19 + signedModulus(hash, 3);
  const eyeSpread = signedModulus(hash, 5);

  return (
    <AvatarFrame size={BEAM_SIZE} id={id} {...props}>
      <g mask={`url(#${id})`}>
        <rect width={BEAM_SIZE} height={BEAM_SIZE} fill={pickColor(hash + 13)} />
        <rect
          x="0"
          y="0"
          width={BEAM_SIZE}
          height={BEAM_SIZE}
          transform={`translate(${wrapperTranslateX} ${wrapperTranslateY}) rotate(${signedModulus(hash, 360)} ${half} ${half}) scale(${1 + signedModulus(hash, BEAM_SIZE / 12) / 10})`}
          fill={wrapperColor}
          rx={isEvenDigit(hash, 1) ? BEAM_SIZE : BEAM_SIZE / 6}
        />
        <g
          transform={`translate(${faceTranslateX} ${faceTranslateY}) rotate(${signedModulus(hash, 10, 3)} ${half} ${half})`}
        >
          {isEvenDigit(hash, 2) ? (
            <path
              d={`M15 ${mouthY}c2 1 4 1 6 0`}
              stroke={faceColor}
              fill="none"
              strokeLinecap="round"
            />
          ) : (
            <path d={`M13,${mouthY} a1,0.75 0 0,0 10,0`} fill={faceColor} />
          )}
          <rect
            x={14 - eyeSpread}
            y={14}
            width={1.5}
            height={2}
            rx={1}
            stroke="none"
            fill={faceColor}
          />
          <rect
            x={20 + eyeSpread}
            y={14}
            width={1.5}
            height={2}
            rx={1}
            stroke="none"
            fill={faceColor}
          />
        </g>
      </g>
    </AvatarFrame>
  );
}

function MarbleAvatar({ hash, id, ...props }: ShapeProps) {
  const half = MARBLE_SIZE / 2;
  const layers = [0, 1, 2].map((index) => {
    const seed = hash * (index + 1);
    return {
      color: pickColor(hash + index),
      translateX: signedModulus(seed, MARBLE_SIZE / 10, 1),
      translateY: signedModulus(seed, MARBLE_SIZE / 10, 2),
      scale: 1.2 + signedModulus(seed, MARBLE_SIZE / 20) / 10,
      rotate: signedModulus(seed, 360, 1),
    };
  });
  // Upstream boring-avatars scales both shapes by the third layer; kept so avatars do not change.
  const scale = layers[2].scale;

  return (
    <AvatarFrame size={MARBLE_SIZE} id={id} {...props}>
      <g mask={`url(#${id})`}>
        <rect width={MARBLE_SIZE} height={MARBLE_SIZE} fill={layers[0].color} />
        <path
          filter={`url(#filter_${id})`}
          d="M32.414 59.35L50.376 70.5H72.5v-71H33.728L26.5 13.381l19.057 27.08L32.414 59.35z"
          fill={layers[1].color}
          transform={`translate(${layers[1].translateX} ${layers[1].translateY}) rotate(${layers[1].rotate} ${half} ${half}) scale(${scale})`}
        />
        <path
          filter={`url(#filter_${id})`}
          style={{ mixBlendMode: "overlay" }}
          d="M22.216 24L0 46.75l14.108 38.129L78 86l-3.081-59.276-22.378 4.005 12.972 20.186-23.35 27.395L22.215 24z"
          fill={layers[2].color}
          transform={`translate(${layers[2].translateX} ${layers[2].translateY}) rotate(${layers[2].rotate} ${half} ${half}) scale(${scale})`}
        />
      </g>
      <defs>
        <filter id={`filter_${id}`} filterUnits="userSpaceOnUse" colorInterpolationFilters="sRGB">
          <feFlood floodOpacity={0} result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation={7} result="effect1_foregroundBlur" />
        </filter>
      </defs>
    </AvatarFrame>
  );
}

function AvatarFrame({
  size,
  id,
  children,
  ...props
}: { size: number; id: string } & React.SVGProps<SVGSVGElement>) {
  return (
    <svg
      viewBox={`0 0 ${size} ${size}`}
      fill="none"
      role="img"
      xmlns="http://www.w3.org/2000/svg"
      width="100%"
      height="100%"
      {...props}
    >
      <mask id={id} maskUnits="userSpaceOnUse" x={0} y={0} width={size} height={size}>
        <rect width={size} height={size} rx={size * 2} fill="#FFFFFF" />
      </mask>
      {children}
    </svg>
  );
}

function hashName(name: string) {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = (hash << 5) - hash + name.charCodeAt(i);
    hash &= hash;
  }
  return Math.abs(hash);
}

function digitAt(value: number, position: number) {
  return Math.floor((value / 10 ** position) % 10);
}

function isEvenDigit(value: number, position: number) {
  return digitAt(value, position) % 2 === 0;
}

function signedModulus(value: number, max: number, signDigit?: number) {
  const remainder = value % max;
  return signDigit && digitAt(value, signDigit) % 2 === 0 ? -remainder : remainder;
}

function pickColor(value: number) {
  return PALETTE[value % PALETTE.length];
}

function contrastColor(hex: string) {
  const red = Number.parseInt(hex.slice(1, 3), 16);
  const green = Number.parseInt(hex.slice(3, 5), 16);
  const blue = Number.parseInt(hex.slice(5, 7), 16);
  return (red * 299 + green * 587 + blue * 114) / 1000 >= 128 ? "#000000" : "#FFFFFF";
}
