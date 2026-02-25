// SPDX-License-Identifier: MPL-2.0

import type { ECharts } from "echarts/core";
import * as echarts from "echarts/core";

// Helper function to get computed CSS value
export const getComputedCssValue = (cssVar: string): string => {
  const root = document.documentElement;
  const computedValue = getComputedStyle(root).getPropertyValue(cssVar).trim();
  return computedValue || `var(${cssVar})`;
};

export const getChartFont = (): string => {
  if (typeof window === "undefined") return "sans-serif";
  return getComputedCssValue("--shaper-font") || "sans-serif";
};

export const getDisplayFont = (): string => {
  if (typeof window === "undefined") return "sans-serif";
  return getComputedCssValue("--shaper-display-font") || "sans-serif";
};

// Function to get theme-appropriate colors
export const getThemeColors = (isDark: boolean) => {
  if (isDark) {
    return {
      primaryColor: getComputedCssValue("--shaper-dark-mode-primary-color-alternate"),
      backgroundColor: getComputedCssValue(
        "--shaper-dark-mode-background-color",
      ),
      backgroundColorSecondary: getComputedCssValue(
        "--shaper-dark-mode-background-color-secondary",
      ),
      borderColor: getComputedCssValue("--shaper-dark-mode-border-color"),
      textColor: getComputedCssValue("--shaper-dark-mode-text-color"),
      textColorSecondary: getComputedCssValue(
        "--shaper-dark-mode-text-color-secondary",
      ),
      referenceLineColor: getComputedCssValue("--shaper-reference-line-color"),
    };
  } else {
    return {
      primaryColor: getComputedCssValue("--shaper-primary-color-alternate"),
      backgroundColor: getComputedCssValue("--shaper-background-color"),
      backgroundColorSecondary: getComputedCssValue(
        "--shaper-background-color-secondary",
      ),
      borderColor: getComputedCssValue("--shaper-border-color"),
      textColor: getComputedCssValue("--shaper-text-color"),
      textColorSecondary: getComputedCssValue("--shaper-text-color-secondary"),
      referenceLineColor: getComputedCssValue("--shaper-reference-line-color"),
    };
  }
};

// Function to download chart as image
export const downloadChartAsImage = async (
  chartInstance: ECharts,
  isDarkMode: boolean,
  chartId: string,
  label?: string,
): Promise<void> => {
  // Get the current chart's dimensions and options
  const chartDom = chartInstance.getDom();
  const { width, height } = chartDom.getBoundingClientRect();
  const chartOptions = chartInstance.getOption() as any;
  chartOptions.animation = false;

  // Check for watermark URL in CSS variables
  const watermarkUrlRaw = getComputedCssValue("--shaper-watermark-url");
  let watermarkUrl = "";
  if (
    watermarkUrlRaw &&
    watermarkUrlRaw !== "none" &&
    watermarkUrlRaw !== "var(--shaper-watermark-url)"
  ) {
    const match = watermarkUrlRaw.match(/url\(['"]?(.*?)['"]?\)/);
    watermarkUrl = match ? match[1] : watermarkUrlRaw;
  }

  const watermarkGap = 10;
  const watermarkHeight = 17;
  const watermarkPadding = 5;
  const newHeight = height + watermarkGap * 2 + watermarkHeight + watermarkPadding;

  if (watermarkUrl) {
    // Adjust grid to keep it at the same position from top
    if (chartOptions.grid) {
      const grids = Array.isArray(chartOptions.grid)
        ? chartOptions.grid
        : [chartOptions.grid];
      const spaceBottom = watermarkGap + watermarkHeight + watermarkPadding;
      grids.forEach((grid: any) => {
        if (grid.bottom !== undefined) {
          if (typeof grid.bottom === "number") {
            grid.bottom += spaceBottom;
          } else if (
            typeof grid.bottom === "string" &&
            grid.bottom.endsWith("px")
          ) {
            grid.bottom = `${parseFloat(grid.bottom) + spaceBottom}px`;
          }
        } else {
          grid.bottom = spaceBottom;
        }
        if (grid.top !== undefined) {
          if (typeof grid.top === "number") {
            grid.top += watermarkGap;
          } else if (
            typeof grid.top === "string" &&
            grid.top.endsWith("px")
          ) {
            grid.top = `${parseFloat(grid.top) + watermarkGap}px`;
          }
        } else {
          grid.top = watermarkGap;
        }
      });
    }
    if (chartOptions.legend) {
      const legends = Array.isArray(chartOptions.legend)
        ? chartOptions.legend
        : [chartOptions.legend];
      legends.forEach((legend: any) => {
        if (legend.top !== undefined) {
          if (typeof legend.top === "number") {
            legend.top += watermarkGap;
          } else if (
            typeof legend.top === "string" &&
            legend.top.endsWith("px")
          ) {
            legend.top = `${parseFloat(legend.top) + watermarkGap}px`;
          }
        } else {
          legend.top = watermarkGap;
        }
      });
    }

    // Pre-load image to ensure it's rendered in getDataURL and to get its dimensions
    let imgWidth = 0;
    await new Promise((resolve) => {
      const img = new Image();
      img.onload = () => {
        const aspectRatio = img.naturalWidth / img.naturalHeight;
        imgWidth = watermarkHeight * aspectRatio;
        resolve(null);
      };
      img.onerror = () => resolve(null);
      img.src = watermarkUrl;
      setTimeout(() => resolve(null), 2000); // Timeout after 2 seconds
    });

    // Add watermark graphic
    const watermarkGraphic = {
      type: "image",
      right: watermarkPadding,
      bottom: watermarkPadding,
      style: {
        image: watermarkUrl,
        height: watermarkHeight,
        width: imgWidth || undefined,
      },
      z: 1000,
    };

    if (!chartOptions.graphic) {
      chartOptions.graphic = [watermarkGraphic];
    } else if (Array.isArray(chartOptions.graphic)) {
      if (chartOptions.graphic.length === 1 && Array.isArray(chartOptions.graphic[0].elements)) {
        chartOptions.graphic[0].elements.push(watermarkGraphic);
      } else {
        chartOptions.graphic = [...chartOptions.graphic, watermarkGraphic];
      }
    } else {
      chartOptions.graphic = [chartOptions.graphic, watermarkGraphic];
    }
  }

  // Create a temporary container for the canvas chart
  const tempContainer = document.createElement("div");
  tempContainer.style.position = "absolute";
  tempContainer.style.top = "-9999px";
  tempContainer.style.left = "-9999px";
  tempContainer.style.width = `${width}px`;
  tempContainer.style.height = `${newHeight}px`;
  document.body.appendChild(tempContainer);

  // Create a temporary chart with canvas renderer
  const tempChart = echarts.init(tempContainer, null, {
    renderer: "canvas",
    width: width,
    height: newHeight,
  });

  // Apply the same options to the temporary chart
  tempChart.setOption(chartOptions);

  try {
    const url = tempChart.getDataURL({
      type: "png",
      pixelRatio: 2,
      backgroundColor: getThemeColors(isDarkMode).backgroundColorSecondary,
    });

    const link = document.createElement("a");

    // Generate a simple filename
    const timestamp = new Date().toISOString().slice(0, 10); // YYYY-MM-DD
    const filename = label
      ? `${label.replace(/[^a-z0-9]/gi, "_").toLowerCase()}-${timestamp}.png`
      : `chart-${chartId}-${timestamp}.png`;

    link.download = filename;
    link.href = url;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  } finally {
    // Clean up: dispose the temporary chart and remove the container
    tempChart.dispose();
    document.body.removeChild(tempContainer);
  }
};

// ECharts color utilities
const echartsColors = {
  primary: {
    light: "var(--shaper-primary-color-alternate)",
    dark: "var(--shaper-dark-mode-primary-color-alternate)",
  },
  color2: {
    light: "var(--shaper-color-two)",
    dark: "var(--shaper-color-two)",
  },
  color3: {
    light: "var(--shaper-color-three)",
    dark: "var(--shaper-color-three)",
  },
  color4: {
    light: "var(--shaper-color-four)",
    dark: "var(--shaper-color-four)",
  },
  color5: {
    light: "var(--shaper-color-five)",
    dark: "var(--shaper-color-five)",
  },
  color6: {
    light: "var(--shaper-color-six)",
    dark: "var(--shaper-color-six)",
  },
  color7: {
    light: "var(--shaper-color-seven)",
    dark: "var(--shaper-color-seven)",
  },
} as const;

const echartsColorKeys = Object.keys(echartsColors) as Array<
  keyof typeof echartsColors
>;

export type EChartsColorKey = keyof typeof echartsColors;

export const AvailableEChartsColors: EChartsColorKey[] = Object.keys(
  echartsColors,
) as Array<EChartsColorKey>;

export const constructCategoryColors = (
  categories: string[],
  colorsByCategory: Record<string, string>,
  isDark: boolean,
): Map<string, string> => {
  const categoryColors = new Map<string, string>();
  let customColorCount = 0;
  categories.forEach((category, index) => {
    let color = colorsByCategory[category];
    if (!color) {
      const echartsKey =
        echartsColors[
          echartsColorKeys[(index - customColorCount) % echartsColorKeys.length]
        ];
      const cssVar = echartsKey[isDark ? "dark" : "light"];
      color = getComputedCssValue(cssVar.replace("var(", "").replace(")", ""));
    } else {
      customColorCount += 1;
    }
    categoryColors.set(category, color);
  });
  return categoryColors;
};

export const getEChartsColor = (
  colorKey: EChartsColorKey,
  isDark: boolean = false,
): string => {
  const color = echartsColors[colorKey];
  if (!color) {
    const fallbackVar = isDark
      ? "--shaper-dark-mode-primary-color-alternate"
      : "--shaper-primary-color-alternate";
    return getComputedCssValue(fallbackVar);
  }

  const cssVar = isDark ? color.dark : color.light;
  // Extract the CSS variable name from the var() function
  const varName = cssVar.replace("var(", "").replace(")", "");

  return getComputedCssValue(varName);
};
