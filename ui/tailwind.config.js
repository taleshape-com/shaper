/** @type {import('tailwindcss').Config} */
// (Optional) Import default theme when using a custom font (Step 7)
import defaultTheme from 'tailwindcss/defaultTheme';
import formPlugin from '@tailwindcss/forms';
import { scopedPreflightStyles, isolateInsideOfContainer } from 'tailwindcss-scoped-preflight';
import containerQueries from '@tailwindcss/container-queries';

export default {
  content: [
    "ui/index.html",
    "ui/src/**/*.{js,ts,jsx,tsx}",
    "server/web/view.html",
  ],
  important: '.shaper-scope',
  theme: {
    extend: {
      colors: {
        "cprimary": "var(--shaper-primary-color)",
        "cprimarya": "var(--shaper-primary-color-alternate)",
        "ctwo": "var(--shaper-color-two)",
        "cthree": "var(--shaper-color-three)",
        "cfour": "var(--shaper-color-four)",
        "cfive": "var(--shaper-color-five)",
        "csix": "var(--shaper-color-six)",
        "cseven": "var(--shaper-color-seven)",
        "ceight": "var(--shaper-color-eight)",
        "cnine": "var(--shaper-color-nine)",
        "cten": "var(--shaper-color-ten)",
        "ctext": "var(--shaper-text-color)",
        "ctext2": "var(--shaper-text-color-secondary)",
        "ctexti": "var(--shaper-text-color-invert)",
        "ctextb": "var(--shaper-text-color-button)",
        "cbg": "var(--shaper-background-color)",
        "cbgs": "var(--shaper-background-color-secondary)",
        "cbga": "var(--shaper-background-color-alternate)",
        "cbgi": "var(--shaper-background-color-invert)",
        "cb": "var(--shaper-border-color)",
        "cerr": "var(--shaper-error-color)",
        "cerra": "var(--shaper-error-color-alternate)",
        "dprimary": "var(--shaper-dark-mode-primary-color)",
        "dprimarya": "var(--shaper-dark-mode-primary-color-alternate)",
        "dtext": "var(--shaper-dark-mode-text-color)",
        "dtext2": "var(--shaper-dark-mode-text-color-secondary)",
        "dtexti": "var(--shaper-dark-mode-text-color-invert)",
        "dtextb": "var(--shaper-dark-mode-text-color-button)",
        "dbg": "var(--shaper-dark-mode-background-color)",
        "dbgs": "var(--shaper-dark-mode-background-color-secondary)",
        "dbga": "var(--shaper-dark-mode-background-color-alternate)",
        "dbgi": "var(--shaper-dark-mode-background-color-invert)",
        "db": "var(--shaper-dark-mode-border-color)",
        "derr": "var(--shaper-dark-mode-error-color)",
        "derra": "var(--shaper-dark-mode-error-color-alternate)",
      },
      fontFamily: {
        sans: ['var(--shaper-font)', ...defaultTheme.fontFamily.sans],
        display: ['var(--shaper-display-font)', ...defaultTheme.fontFamily.sans],
      },
      containers: {
        'sm': '640px',
        'lg': '1024px',
        'xl': '1280px',
        '2xl': '1536px',
        '4xl': '1948px',
      },
      keyframes: {
        hide: {
          from: { opacity: "1" },
          to: { opacity: "0" },
        },
        slideDownAndFade: {
          from: { opacity: "0", transform: "translateY(-6px)" },
          to: { opacity: "1", transform: "translateY(0)" },
        },
        slideLeftAndFade: {
          from: { opacity: "0", transform: "translateX(6px)" },
          to: { opacity: "1", transform: "translateX(0)" },
        },
        slideUpAndFade: {
          from: { opacity: "0", transform: "translateY(6px)" },
          to: { opacity: "1", transform: "translateY(0)" },
        },
        slideRightAndFade: {
          from: { opacity: "0", transform: "translateX(-6px)" },
          to: { opacity: "1", transform: "translateX(0)" },
        },
        accordionOpen: {
          from: { height: "0px" },
          to: { height: "var(--radix-accordion-content-height)" },
        },
        accordionClose: {
          from: {
            height: "var(--radix-accordion-content-height)",
          },
          to: { height: "0px" },
        },
        dialogOverlayShow: {
          from: { opacity: "0" },
          to: { opacity: "1" },
        },
        dialogContentShow: {
          from: {
            opacity: "0",
            transform: "translate(-50%, -45%) scale(0.95)",
          },
          to: { opacity: "1", transform: "translate(-50%, -50%) scale(1)" },
        },
        drawerSlideLeftAndFade: {
          from: { opacity: "0", transform: "translateX(100%)" },
          to: { opacity: "1", transform: "translateX(0)" },
        },
        drawerSlideRightAndFade: {
          from: { opacity: "1", transform: "translateX(0)" },
          to: { opacity: "0", transform: "translateX(100%)" },
        },
      },
      animation: {
        hide: "hide 150ms cubic-bezier(0.16, 1, 0.3, 1)",
        slideDownAndFade:
          "slideDownAndFade 150ms cubic-bezier(0.16, 1, 0.3, 1)",
        slideLeftAndFade:
          "slideLeftAndFade 150ms cubic-bezier(0.16, 1, 0.3, 1)",
        slideUpAndFade: "slideUpAndFade 150ms cubic-bezier(0.16, 1, 0.3, 1)",
        slideRightAndFade:
          "slideRightAndFade 150ms cubic-bezier(0.16, 1, 0.3, 1)",
        // Accordion
        accordionOpen: "accordionOpen 150ms cubic-bezier(0.87, 0, 0.13, 1)",
        accordionClose: "accordionClose 150ms cubic-bezier(0.87, 0, 0.13, 1)",
        // Dialog
        dialogOverlayShow:
          "dialogOverlayShow 150ms cubic-bezier(0.16, 1, 0.3, 1)",
        dialogContentShow:
          "dialogContentShow 150ms cubic-bezier(0.16, 1, 0.3, 1)",
        // Drawer
        drawerSlideLeftAndFade:
          "drawerSlideLeftAndFade 150ms cubic-bezier(0.16, 1, 0.3, 1)",
        drawerSlideRightAndFade: "drawerSlideRightAndFade 150ms ease-in",
      },
    },
  },
  plugins: [
    formPlugin,
    containerQueries,
    scopedPreflightStyles({
      isolationStrategy: isolateInsideOfContainer('.shaper-scope'),
    }),
  ],
}
