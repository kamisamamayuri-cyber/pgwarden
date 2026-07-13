import type { Config } from "tailwindcss";
import daisyui from "daisyui";
import * as daisyuiThemes from "daisyui/src/theming/themes";

const primaryRed = "#FF001F";
const primaryDark = "#1E1E1E";
const primaryLight = "#FFFFFF";

export default {
  content: [
    "./internal/view/web/**/*.go",
    "./internal/view/static/js/init-dialogs.js",
  ],
  plugins: [daisyui as any],
  daisyui: {
    logs: false,
    themes: [
      {
        light: {
          ...daisyuiThemes.light,
          primary: primaryRed,
          "primary-content": primaryLight,
          secondary: primaryDark,
          "secondary-content": primaryLight,
          accent: primaryRed,
          neutral: primaryDark,
          "neutral-content": primaryLight,
          "base-100": primaryLight,
          "base-200": "#F4F4F4",
          "base-300": "#E8E8E8",
          "base-content": primaryDark,
          info: "#2563EB",
          success: "#15803D",
          warning: "#CA8A04",
          error: primaryRed,
          "success-content": primaryLight,
          "error-content": primaryLight,
        },
        dark: {
          ...daisyuiThemes.dark,
          primary: primaryRed,
          "primary-content": primaryLight,
          secondary: "#2A2A2A",
          "secondary-content": primaryLight,
          accent: primaryRed,
          neutral: "#2A2A2A",
          "neutral-content": primaryLight,
          "base-100": primaryDark,
          "base-200": "#2A2A2A",
          "base-300": "#333333",
          "base-content": primaryLight,
          info: "#60A5FA",
          success: "#4ADE80",
          warning: "#FACC15",
          error: "#FF4D6A",
          "success-content": primaryDark,
          "error-content": primaryLight,
        },
      },
    ],
    darkTheme: "dark",
  },
  theme: {
    screens: {
      desk: "768px", // only one breakpoint to keep it simple
    },
    extend: {
      colors: {
        pgwarden: {
          red: primaryRed,
          graphite: primaryDark,
          white: primaryLight,
        },
      },
    },
  },
} satisfies Config;
