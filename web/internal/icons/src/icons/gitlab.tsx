import { type IconProps, sizeMap } from "../props";

export function Gitlab({ iconSize = "xl-thin", ...props }: IconProps) {
  const { iconSize: pixelSize } = sizeMap[iconSize];

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={pixelSize}
      height={pixelSize}
      viewBox="0 0 32 32"
      {...props}
    >
      <g fill="currentColor">
        <path d="M31.94,18.116l-1.789-5.513-3.552-10.918c-.183-.563-.973-.563-1.156,0l-3.552,10.918H10.109L6.557,1.685c-.183-.563-.973-.563-1.156,0L1.849,12.603,.06,18.116c-.164,.503,.016,1.055,.443,1.366l15.497,11.259,15.497-11.259c.427-.311,.607-.863,.443-1.366Z" />
      </g>
    </svg>
  );
}
