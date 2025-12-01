/** @type {import('tailwindcss').Config} */
export default {
  content: [],
  theme: {
    extend: {
      fontFamily: {
        sans: ["var(--font-geist-sans)"],
        mono: ["var(--font-geist-mono)"],
      },
      colors: {
        black: "black",
        white: "white",
        transparent: "transparent",
        current: "currentColor",
        ...generateRadixColors(),
      },
      dropShadow: {
        // from vitor's figma
        button: "0px 0px 0px 3px hsl(var(--grayA-6))",
      },
      opacity: {
        5: "0.05",
        10: "0.1",
        15: "0.15",
        20: "0.2",
        25: "0.25",
        30: "0.3",
        35: "0.35",
        40: "0.4",
        45: "0.45",
        50: "0.5",
        55: "0.55",
        60: "0.6",
        65: "0.65",
        70: "0.7",
        75: "0.75",
        80: "0.8",
        85: "0.85",
        90: "0.9",
        95: "0.95",
        98: "0.98",
      },
      keyframes: {
        marquee: {
          "0%": { transform: "translateX(0%)" },
          "100%": { transform: "translateX(-100%)" },
        },
      },
      animation: {
        marquee: "marquee 15s linear infinite",
      },
    },
  },
  plugins: [],
};

const getColor = (colorVar, { opacityVariable, opacityValue }) => {
  // For alpha colors, we need to extract the alpha from the variable itself
  // to avoid the syntax error in the generated CSS
  const alphaColors = [
    "grayA",
    "errorA",
    "successA",
    "warningA",
    "orangeA",
    "infoA",
    "redA",
    "grassA",
    "blueA",
    "featureA",
    "blackA",
    "whiteA",
  ];
  if (alphaColors.some((color) => colorVar.includes(color))) {
    return `hsla(var(--${colorVar.replace("--", "")}))`;
  }
  if (opacityValue !== undefined) {
    return `hsla(var(${colorVar}), ${opacityValue})`;
  }
  if (opacityVariable !== undefined) {
    return `hsla(var(${colorVar}), var(${opacityVariable}, 1))`;
  }
  return `hsl(var(${colorVar}))`;
};

function generateRadixColors() {
  const colorNames = [
    "gray",
    "grayA",
    "info",
    "infoA",
    "success",
    "successA",
    "orange",
    "orangeA",
    "warning",
    "warningA",
    "error",
    "errorA",
    "feature",
    "featureA",
    "accent",
    "base",
    "red",
    "redA",
    "grass",
    "grassA",
    "blue",
    "blueA",
    "blackA",
    "whiteA",
    "cyan",
    "bronze",
  ];
  const colors = {};
  colorNames.forEach((name) => {
    colors[name] = {};
    for (let i = 1; i <= 12; i++) {
      colors[name][i] = (params) => getColor(`--${name}-${i}`, params);
    }
  });
  return colors;
}
