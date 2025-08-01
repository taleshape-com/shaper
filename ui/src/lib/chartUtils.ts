// SPDX-License-Identifier: MPL-2.0

// Helper function to get computed CSS value
export const getComputedCssValue = (cssVar: string): string => {
  const root = document.documentElement;
  const computedValue = getComputedStyle(root).getPropertyValue(cssVar).trim();
  return computedValue || `var(${cssVar})`;
};

// Helper function to get the display font
export const getChartFont = (): string => {
  if (typeof window === 'undefined') return 'sans-serif';
  return getComputedCssValue('--shaper-font') || 'sans-serif';
};

// Function to get theme-appropriate colors
export const getThemeColors = (isDark: boolean) => {
  if (isDark) {
    return {
      primaryColor: getComputedCssValue('--shaper-dark-mode-primary-color'),
      backgroundColor: getComputedCssValue('--shaper-dark-mode-background-color'),
      borderColor: getComputedCssValue('--shaper-dark-mode-border-color'),
      textColor: getComputedCssValue('--shaper-dark-mode-text-color'),
      textColorSecondary: getComputedCssValue('--shaper-dark-mode-text-color-secondary'),
      referenceLineColor: getComputedCssValue('--shaper-reference-line-color'),
    };
  } else {
    return {
      primaryColor: getComputedCssValue('--shaper-primary-color'),
      backgroundColor: getComputedCssValue('--shaper-background-color'),
      borderColor: getComputedCssValue('--shaper-border-color'),
      textColor: getComputedCssValue('--shaper-text-color'),
      textColorSecondary: getComputedCssValue('--shaper-text-color-secondary'),
      referenceLineColor: getComputedCssValue('--shaper-reference-line-color'),
    };
  }
};

// Function to download chart as image
export const downloadChartAsImage = (
  chartInstance: echarts.ECharts,
  chartId: string,
  label?: string
): void => {
  const url = chartInstance.getDataURL({
    type: 'png',
    pixelRatio: 2,
    backgroundColor: getThemeColors(false).backgroundColor,
  });

  const link = document.createElement('a');

  // Generate a simple filename
  const timestamp = new Date().toISOString().slice(0, 10); // YYYY-MM-DD
  const filename = label
    ? `${label.replace(/[^a-z0-9]/gi, '_').toLowerCase()}-${timestamp}.png`
    : `chart-${chartId}-${timestamp}.png`;

  link.download = filename;
  link.href = url;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
};

// ECharts color utilities
export const echartsColors = {
  primary: {
    light: 'var(--shaper-primary-color)',
    dark: 'var(--shaper-dark-mode-primary-color)',
  },
  color2: {
    light: 'var(--shaper-color-two)',
    dark: 'var(--shaper-color-two)',
  },
  color3: {
    light: 'var(--shaper-color-three)',
    dark: 'var(--shaper-color-three)',
  },
  color4: {
    light: 'var(--shaper-color-four)',
    dark: 'var(--shaper-color-four)',
  },
  color5: {
    light: 'var(--shaper-color-five)',
    dark: 'var(--shaper-color-five)',
  },
  color6: {
    light: 'var(--shaper-color-six)',
    dark: 'var(--shaper-color-six)',
  },
  color7: {
    light: 'var(--shaper-color-seven)',
    dark: 'var(--shaper-color-seven)',
  },
  color8: {
    light: 'var(--shaper-color-eight)',
    dark: 'var(--shaper-color-eight)',
  },
  color9: {
    light: 'var(--shaper-color-nine)',
    dark: 'var(--shaper-color-nine)',
  },
  color10: {
    light: 'var(--shaper-color-ten)',
    dark: 'var(--shaper-color-ten)',
  },
} as const;

export type EChartsColorKey = keyof typeof echartsColors;

export const AvailableEChartsColors: EChartsColorKey[] = Object.keys(
  echartsColors,
) as Array<EChartsColorKey>;

export const constructEChartsCategoryColors = (
  categories: string[],
  colors: EChartsColorKey[],
): Map<string, EChartsColorKey> => {
  const categoryColors = new Map<string, EChartsColorKey>();
  categories.forEach((category, index) => {
    categoryColors.set(category, colors[index % colors.length]);
  });
  return categoryColors;
};

export const getEChartsColor = (
  colorKey: EChartsColorKey,
  isDark: boolean = false,
): string => {
  const color = echartsColors[colorKey];
  if (!color) {
    const fallbackVar = isDark ? '--shaper-dark-mode-primary-color' : '--shaper-primary-color';
    return getComputedCssValue(fallbackVar);
  }

  const cssVar = isDark ? color.dark : color.light;
  // Extract the CSS variable name from the var() function
  const varName = cssVar.replace('var(', '').replace(')', '');

  return getComputedCssValue(varName);
};
