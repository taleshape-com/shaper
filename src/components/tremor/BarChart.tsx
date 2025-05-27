import React from "react";
import { RiArrowLeftSLine, RiArrowRightSLine } from "@remixicon/react";
import {
  Bar,
  CartesianGrid,
  Label,
  BarChart as RechartsBarChart,
  Legend as RechartsLegend,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { AxisDomain } from "recharts/types/util/types";
import { ChartDownloadButton } from "./ChartDownloadButton";

import {
  AvailableChartColors,
  AvailableChartColorsKeys,
  constructCategoryColors,
  getColorClassName,
  getYAxisDomain,
} from "../../lib/chartUtils";
import { useOnWindowResize } from "../../hooks/useOnWindowResize";
import { cx, getNameIfSet } from "../../lib/utils";
import { ChartHoverContext } from "../../contexts/ChartHoverContext";
import { Column } from "../../lib/dashboard";
import { formatValue } from "../../lib/render";
import { translate } from "../../lib/translate";

//#region Shape

function deepEqual<T>(obj1: T, obj2: T): boolean {
  if (obj1 === obj2) return true;

  if (
    typeof obj1 !== "object" ||
    typeof obj2 !== "object" ||
    obj1 === null ||
    obj2 === null
  ) {
    return false;
  }

  const keys1 = Object.keys(obj1) as Array<keyof T>;
  const keys2 = Object.keys(obj2) as Array<keyof T>;

  if (keys1.length !== keys2.length) return false;

  for (const key of keys1) {
    if (!keys2.includes(key) || !deepEqual(obj1[key], obj2[key])) return false;
  }

  return true;
}

const renderShape = (
  props: any,
  activeBar: any | undefined,
  activeLegend: string | undefined,
  layout: string,
) => {
  const { fillOpacity, name, payload, value } = props;
  let { x, width, y, height } = props;

  if (layout === "horizontal" && height < 0) {
    y += height;
    height = Math.abs(height); // height must be a positive number
  } else if (layout === "vertical" && width < 0) {
    x += width;
    width = Math.abs(width); // width must be a positive number
  }

  return (
    <rect
      x={x}
      y={y}
      width={width}
      height={height}
      opacity={
        activeBar || (activeLegend && activeLegend !== name)
          ? deepEqual(activeBar, { ...payload, value })
            ? fillOpacity
            : 0.3
          : fillOpacity
      }
    />
  );
};

//#region Legend

interface LegendItemProps {
  name: string;
  color: AvailableChartColorsKeys;
  onClick?: (name: string, color: AvailableChartColorsKeys) => void;
  activeLegend?: string;
}

const LegendItem = ({
  name,
  color,
  onClick,
  activeLegend,
}: LegendItemProps) => {
  const hasOnValueChange = !!onClick;
  const textWidth = React.useMemo(() => {
    // Create a temporary SVG to measure text width
    const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    const text = document.createElementNS("http://www.w3.org/2000/svg", "text");
    text.setAttribute("class", "text-xs");
    text.textContent = name;
    svg.appendChild(text);
    document.body.appendChild(svg);
    const width = text.getComputedTextLength();
    document.body.removeChild(svg);
    return width;
  }, [name]);

  return (
    <li
      className={cx(
        // base
        "group inline-block rounded px-2 py-1.5 transition",
        hasOnValueChange
          ? "cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800"
          : "cursor-default",
      )}
      onClick={(e) => {
        e.stopPropagation();
        onClick?.(name, color);
      }}
    >
      <span
        className={cx(
          "inline-block size-2 rounded-sm mr-1.5",
          getColorClassName(color, "bg"),
          activeLegend && activeLegend !== name ? "opacity-40" : "opacity-100",
        )}
        aria-hidden={true}
      />
      <svg
        className={cx(
          "inline-block",
          activeLegend && activeLegend !== name ? "opacity-40" : "opacity-100",
        )}
        height="16"
        width={textWidth + 2}
        style={{ verticalAlign: "middle" }}
      >
        <text
          x="0"
          y="12"
          className={cx(
            "text-xs fill-ctext dark:fill-dtext",
            hasOnValueChange &&
            "group-hover:fill-gray-900 dark:group-hover:fill-gray-50",
          )}
        >
          {name}
        </text>
      </svg>
    </li>
  );
};

interface ScrollButtonProps {
  icon: React.ElementType;
  onClick?: () => void;
  disabled?: boolean;
}

const ScrollButton = ({ icon, onClick, disabled }: ScrollButtonProps) => {
  const Icon = icon;
  const [isPressed, setIsPressed] = React.useState(false);
  const intervalRef = React.useRef<NodeJS.Timeout | null>(null);

  React.useEffect(() => {
    if (isPressed) {
      intervalRef.current = setInterval(() => {
        onClick?.();
      }, 300);
    } else {
      clearInterval(intervalRef.current as NodeJS.Timeout);
    }
    return () => clearInterval(intervalRef.current as NodeJS.Timeout);
  }, [isPressed, onClick]);

  React.useEffect(() => {
    if (disabled) {
      clearInterval(intervalRef.current as NodeJS.Timeout);
      setIsPressed(false);
    }
  }, [disabled]);

  return (
    <button
      type="button"
      className={cx(
        // base
        "group inline-flex size-5 items-center truncate rounded transition",
        disabled
          ? "cursor-not-allowed text-gray-400 dark:text-gray-600"
          : "cursor-pointer text-ctext hover:bg-cbga dark:text-dtext dark:hover:bg-dbga",
      )}
      disabled={disabled}
      onClick={(e) => {
        e.stopPropagation();
        onClick?.();
      }}
      onMouseDown={(e) => {
        e.stopPropagation();
        setIsPressed(true);
      }}
      onMouseUp={(e) => {
        e.stopPropagation();
        setIsPressed(false);
      }}
    >
      <Icon className="size-full" aria-hidden="true" />
    </button>
  );
};

interface LegendProps extends React.OlHTMLAttributes<HTMLOListElement> {
  categories: string[];
  colors?: AvailableChartColorsKeys[];
  onClickLegendItem?: (category: string, color: string) => void;
  activeLegend?: string;
  enableLegendSlider?: boolean;
}

type HasScrollProps = {
  left: boolean;
  right: boolean;
};

const Legend = React.forwardRef<HTMLOListElement, LegendProps>((props, ref) => {
  const {
    categories,
    colors = AvailableChartColors,
    className,
    onClickLegendItem,
    activeLegend,
    enableLegendSlider = false,
    ...other
  } = props;
  const scrollableRef = React.useRef<HTMLInputElement>(null);
  const scrollButtonsRef = React.useRef<HTMLDivElement>(null);
  const [hasScroll, setHasScroll] = React.useState<HasScrollProps>({
    left: false,
    right: categories.length > 5,
  });
  const [isKeyDowned, setIsKeyDowned] = React.useState<string | null>(null);
  const intervalRef = React.useRef<NodeJS.Timeout | null>(null);

  const checkScroll = React.useCallback(() => {
    const scrollable = scrollableRef?.current;
    if (!scrollable) return;

    const hasLeftScroll = scrollable.scrollLeft > 0;
    const hasRightScroll =
      scrollable.scrollWidth - scrollable.clientWidth > scrollable.scrollLeft;

    setHasScroll({ left: hasLeftScroll, right: hasRightScroll });
  }, [setHasScroll]);

  const scrollToTest = React.useCallback(
    (direction: "left" | "right") => {
      const element = scrollableRef?.current;
      const scrollButtons = scrollButtonsRef?.current;
      const scrollButtonsWith = scrollButtons?.clientWidth ?? 0;
      const width = element?.clientWidth ?? 0;

      if (element && enableLegendSlider) {
        element.scrollTo({
          left:
            direction === "left"
              ? element.scrollLeft - width + scrollButtonsWith
              : element.scrollLeft + width - scrollButtonsWith,
          behavior: "smooth",
        });
        setTimeout(() => {
          checkScroll();
        }, 400);
      }
    },
    [enableLegendSlider, checkScroll],
  );

  React.useEffect(() => {
    const keyDownHandler = (key: string) => {
      if (key === "ArrowLeft") {
        scrollToTest("left");
      } else if (key === "ArrowRight") {
        scrollToTest("right");
      }
    };
    if (isKeyDowned) {
      keyDownHandler(isKeyDowned);
      intervalRef.current = setInterval(() => {
        keyDownHandler(isKeyDowned);
      }, 300);
    } else {
      clearInterval(intervalRef.current as NodeJS.Timeout);
    }
    return () => clearInterval(intervalRef.current as NodeJS.Timeout);
  }, [isKeyDowned, scrollToTest]);

  const keyDown = (e: KeyboardEvent) => {
    e.stopPropagation();
    if (e.key === "ArrowLeft" || e.key === "ArrowRight") {
      e.preventDefault();
      setIsKeyDowned(e.key);
    }
  };
  const keyUp = (e: KeyboardEvent) => {
    e.stopPropagation();
    setIsKeyDowned(null);
  };

  React.useEffect(() => {
    const scrollable = scrollableRef?.current;
    if (enableLegendSlider) {
      checkScroll();
      scrollable?.addEventListener("keydown", keyDown);
      scrollable?.addEventListener("keyup", keyUp);
    }

    return () => {
      scrollable?.removeEventListener("keydown", keyDown);
      scrollable?.removeEventListener("keyup", keyUp);
    };
  }, [checkScroll, enableLegendSlider]);

  return (
    <ol
      ref={ref}
      className={cx("relative overflow-hidden", className)}
      {...other}
    >
      <div
        ref={scrollableRef}
        tabIndex={0}
        className={cx(
          "flex h-full",
          enableLegendSlider
            ? hasScroll?.right || hasScroll?.left
              ? "snap-mandatory items-center overflow-auto pl-4 pr-12 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
              : ""
            : "flex-wrap",
        )}
      >
        {categories.map((category, index) => (
          <LegendItem
            key={`item-${index}`}
            name={category}
            color={colors[index] as AvailableChartColorsKeys}
            onClick={onClickLegendItem}
            activeLegend={activeLegend}
          />
        ))}
      </div>
      {enableLegendSlider && (hasScroll?.right || hasScroll?.left) ? (
        <>
          <div
            className={cx(
              // base
              "absolute bottom-0 right-0 top-0 flex h-full items-center justify-center pr-1",
              // background color
              "bg-cbg dark:bg-dbg",
            )}
          >
            <ScrollButton
              icon={RiArrowLeftSLine}
              onClick={() => {
                setIsKeyDowned(null);
                scrollToTest("left");
              }}
              disabled={!hasScroll?.left}
            />
            <ScrollButton
              icon={RiArrowRightSLine}
              onClick={() => {
                setIsKeyDowned(null);
                scrollToTest("right");
              }}
              disabled={!hasScroll?.right}
            />
          </div>
        </>
      ) : null}
    </ol>
  );
});

Legend.displayName = "Legend";

const ChartLegend = (
  { payload }: any,
  categoryColors: Map<string, AvailableChartColorsKeys>,
  setLegendHeight: React.Dispatch<React.SetStateAction<number>>,
  activeLegend: string | undefined,
  onClick?: (category: string, color: string) => void,
  enableLegendSlider?: boolean,
) => {
  const legendRef = React.useRef<HTMLDivElement>(null);

  useOnWindowResize(() => {
    const calculateHeight = (height: number | undefined) =>
      height ? Number(height) + 15 : 60;
    setLegendHeight(calculateHeight(legendRef.current?.clientHeight));
  });

  const filteredPayload = payload.filter((item: any) => item.type !== "none");

  return (
    <div
      ref={legendRef}
      className={cx(
        "flex items-center justify-end"
      )}
    >
      <Legend
        categories={filteredPayload.map((entry: any) => entry.value)}
        colors={filteredPayload.map((entry: any) =>
          categoryColors.get(entry.value),
        )}
        onClickLegendItem={onClick}
        activeLegend={activeLegend}
        enableLegendSlider={enableLegendSlider}
      />
    </div>
  );
};

//#region Tooltip

type TooltipProps = Pick<ChartTooltipProps, "active" | "payload" | "label">;

type PayloadItem = {
  category: string;
  value: number;
  index: string;
  color: AvailableChartColorsKeys;
  type?: string;
  payload: any;
};

interface ChartTooltipProps {
  active: boolean | undefined;
  payload: PayloadItem[];
  label: string;
  valueFormatter: (value: number) => string;
  extraData?: Record<string, [any, Column["type"]]>;
  total?: number;
}

const ChartTooltip = ({
  active,
  payload,
  label,
  valueFormatter,
  extraData,
  total,
}: ChartTooltipProps) => {
  if (active && payload && payload.length) {
    return (
      <div
        className={cx(
          // base
          "rounded-md border text-sm shadow-md",
          // border color
          "border-cb dark:border-db",
          // background color
          "bg-cbg dark:bg-dbg",
        )}
      >
        {
          label && (
            <div className={cx("border-b border-inherit px-4 py-2")}>
              <p
                className={cx(
                  // base
                  "font-medium",
                  // text color
                  "text-ctext dark:text-dtext",
                )}
              >
                {label}
              </p>
            </div>
          )
        }
        {total && (
          <div className={cx("border-b border-inherit px-4 py-2")}>
            <p className="flex justify-between space-x-2">
              <span className="font-medium">{translate('Total')}</span>
              <span>{formatValue(total, 'number', true)}</span>
            </p>
          </div>
        )}
        {extraData && (
          <div className={cx("border-b border-inherit px-4 py-2")}>
            {Object.entries(extraData).map(([key, [value, columnType]]) => {
              return (
                <p className="flex justify-between space-x-2">
                  <span className="font-medium">{key}</span>
                  <span>{formatValue(value, columnType, true)}</span>
                </p>
              )
            })}
          </div>
        )}
        <div className={cx("space-y-1 px-4 py-2")}>
          {payload.map(({ value, category, color }, index) => {
            const cat = getNameIfSet(category)
            return (
              <div
                key={`id-${index}`}
                className={cx("flex items-center justify-between", {
                  "space-x-8": cat != null,
                  "space-x-2": cat == null,
                })}
              >
                <div className="flex items-center space-x-2">
                  <span
                    aria-hidden="true"
                    className={cx(
                      "size-2 shrink-0 rounded-sm",
                      getColorClassName(color, "bg"),
                    )}
                  />
                  {cat && (
                    <p
                      className={cx(
                        // base
                        "whitespace-nowrap text-right",
                        // text color
                        "text-ctext dark:text-dtext",
                      )}
                    >
                      {cat}
                    </p>
                  )}
                </div>
                <p
                  className={cx(
                    // base
                    "whitespace-nowrap text-right font-medium tabular-nums",
                    // text color
                    "text-ctext dark:text-dtext",
                  )}
                >
                  {valueFormatter(value)}
                </p>
              </div>
            );
          })}
        </div>
      </div>
    );
  }
  return null;
};

//#region BarChart

type BaseEventProps = {
  eventType: "category" | "bar";
  categoryClicked: string;
  [key: string]: number | string;
};

type BarChartEventProps = BaseEventProps | null | undefined;

interface BarChartProps extends React.HTMLAttributes<HTMLDivElement> {
  chartId: string;
  data: Record<string, any>[];
  extraDataByIndexAxis: Record<string, Record<string, any>>;
  index: string;
  indexType: Column['type'];
  valueType: Column['type'];
  categories: string[];
  valueFormatter?: (value: number) => string;
  indexFormatter?: (value: number) => string;
  startEndOnly?: boolean;
  showXAxis?: boolean;
  showYAxis?: boolean;
  showGridLines?: boolean;
  yAxisWidth?: number;
  intervalType?: "preserveStartEnd" | "equidistantPreserveStart";
  showTooltip?: boolean;
  showLegend?: boolean;
  autoMinValue?: boolean;
  minValue?: number;
  maxValue?: number;
  allowDecimals?: boolean;
  enableLegendSlider?: boolean;
  tickGap?: number;
  barCategoryGap?: string | number;
  xAxisLabel?: string;
  yAxisLabel?: string;
  layout?: "vertical" | "horizontal";
  type?: "default" | "stacked" | "percent";
  indexAxisDomain?: AxisDomain;
  label?: string;
}

const BarChart = React.forwardRef<HTMLDivElement, BarChartProps>(
  (props, forwardedRef) => {
    const {
      data = [],
      extraDataByIndexAxis,
      categories = [],
      index,
      indexType,
      valueType,
      valueFormatter = (value: number) => value.toString(),
      indexFormatter = (value: number) => value.toString(),
      startEndOnly = false,
      showXAxis = true,
      showYAxis = true,
      showGridLines = true,
      yAxisWidth = 56,
      intervalType = "equidistantPreserveStart",
      showTooltip = true,
      showLegend = true,
      autoMinValue = false,
      minValue,
      maxValue,
      allowDecimals = true,
      className,
      enableLegendSlider = false,
      barCategoryGap,
      tickGap = 5,
      xAxisLabel,
      yAxisLabel,
      layout = "horizontal",
      type = "default",
      indexAxisDomain = ["auto", "auto"],
      chartId,
      label,
      ...other
    } = props;
    const paddingValue =
      (!showXAxis && !showYAxis) || (startEndOnly && !showYAxis) ? 0 : 20;
    const [legendHeight, setLegendHeight] = React.useState(60);
    const categoryColors = constructCategoryColors(categories, AvailableChartColors);
    const valueAxisDomain = getYAxisDomain(autoMinValue, minValue, maxValue);
    const stacked = type === "stacked" || type === "percent";

    const { hoveredIndex, hoveredChartId, hoveredIndexType, setHoverState } =
      React.useContext(ChartHoverContext);

    const [chartElement, setChartElement] = React.useState<HTMLDivElement | null>(null);
    const [isChartHovered, setIsChartHovered] = React.useState(false);
    const [isDownloading, setIsDownloading] = React.useState(false);

    // Create a callback ref that handles both refs
    const setRefs = React.useCallback((node: HTMLDivElement | null) => {
      // Set the forwarded ref
      if (typeof forwardedRef === "function") {
        forwardedRef(node);
      } else if (forwardedRef) {
        (forwardedRef as React.MutableRefObject<HTMLDivElement | null>).current = node;
      }
      // Update our state
      setChartElement(node);
    }, [forwardedRef]);

    function valueToPercent(value: number) {
      return `${(value * 100).toFixed(0)}%`;
    }

    const downloadButton = !isDownloading ? (
      <ChartDownloadButton
        chartElement={chartElement}
        chartId={chartId}
        isVisible={isChartHovered}
        onDownloadStart={() => setIsDownloading(true)}
        onDownloadEnd={() => setIsDownloading(false)}
        label={label}
      />
    ) : null;

    return (
      <div
        ref={setRefs}
        className={cx("h-80 w-full relative", className)}
        tremor-id="tremor-raw"
        onMouseEnter={() => setIsChartHovered(true)}
        onMouseLeave={() => setIsChartHovered(false)}
        {...other}
      >
        {downloadButton}
        <ResponsiveContainer>
          <RechartsBarChart
            data={data}
            onMouseMove={(state) => {
              if (state.activeLabel !== undefined) {
                setHoverState(state.activeLabel, chartId, indexType);
              }
            }}
            onMouseLeave={() => {
              if (hoveredChartId === chartId) {
                setHoverState(null, null, null);
              }
            }}
            margin={{
              bottom: xAxisLabel ? 25 : undefined,
              left: yAxisLabel ? 25 : 5,
              right: 5,
              top: 15,
            }}
            stackOffset={type === "percent" ? "expand" : undefined}
            layout={layout}
            barCategoryGap={barCategoryGap}
          >
            {showGridLines ? (
              <CartesianGrid
                className={cx("stroke-cb stroke-1 dark:stroke-db")}
                horizontal={layout !== "vertical"}
                vertical={layout === "vertical"}
              />
            ) : null}
            <XAxis
              hide={!showXAxis}
              tick={{
                transform:
                  layout !== "vertical" ? "translate(0, 6)" : undefined,
              }}
              fill=""
              stroke=""
              className={cx(
                // base
                "text-xs",
                // text fill
                "fill-ctext2 dark:fill-dtext2",
                { "mt-4": layout !== "vertical" },
              )}
              tickLine={false}
              axisLine={false}
              minTickGap={tickGap}
              {...(layout !== "vertical"
                ? {
                  padding: {
                    left: paddingValue,
                    right: paddingValue,
                  },
                  dataKey: index,
                  interval: startEndOnly ? "preserveStartEnd" : intervalType,
                  ticks: startEndOnly
                    ? [data[0][index], data[data.length - 1][index]]
                    : undefined,
                  tickFormatter:
                    type === "percent" ? valueToPercent : indexFormatter,
                  type:
                    Array.isArray(indexAxisDomain) &&
                      indexAxisDomain[0] !== "auto" &&
                      data.length > 7
                      ? "number"
                      : "category",
                  domain: indexAxisDomain,
                }
                : {
                  type: "number",
                  tickFormatter:
                    type === "percent" ? valueToPercent : valueFormatter,
                  allowDecimals: allowDecimals,
                  domain: valueAxisDomain,
                })}
            >
              {xAxisLabel && (
                <Label
                  position="insideBottom"
                  offset={-20}
                  className="fill-ctext text-sm font-medium dark:fill-dtext"
                >
                  {xAxisLabel}
                </Label>
              )}
            </XAxis>
            <YAxis
              width={yAxisWidth}
              hide={!showYAxis}
              axisLine={false}
              tickLine={false}
              fill=""
              stroke=""
              className={cx(
                // base
                "text-xs",
                // text fill
                "fill-ctext2 dark:fill-dtext2",
              )}
              tick={{
                transform:
                  layout !== "vertical"
                    ? "translate(-3, 0)"
                    : "translate(0, 0)",
              }}
              {...(layout !== "vertical"
                ? {
                  type: "number",
                  domain: valueAxisDomain as AxisDomain,
                  tickFormatter:
                    type === "percent" ? valueToPercent : valueFormatter,
                  allowDecimals: allowDecimals,
                }
                : {
                  dataKey: index,
                  ticks: startEndOnly
                    ? [data[0][index], data[data.length - 1][index]]
                    : undefined,
                  type:
                    Array.isArray(indexAxisDomain) &&
                      indexAxisDomain[0] !== "auto" &&
                      data.length > 7
                      ? "number"
                      : "category",
                  interval: "equidistantPreserveStart",
                  tickFormatter:
                    type === "percent" ? valueToPercent : indexFormatter,
                  domain: indexAxisDomain,
                })}
            >
              {yAxisLabel && (
                <Label
                  position="insideLeft"
                  style={{ textAnchor: "middle" }}
                  angle={-90}
                  offset={-15}
                  className="fill-ctext text-sm font-medium dark:fill-dtext"
                >
                  {yAxisLabel}
                </Label>
              )}
            </YAxis>
            <Tooltip
              wrapperStyle={{ outline: "none", zIndex: 30 }}
              isAnimationActive={true}
              animationDuration={100}
              cursor={{
                fill:
                  window.matchMedia &&
                    window.matchMedia("(prefers-color-scheme: dark)").matches
                    ? "var(--shaper-dark-mode-background-color-invert)"
                    : "var(--shaper-background-color-invert)",
                opacity: 0.05,
              }}
              offset={20}
              position={{
                y: layout === "horizontal" ? 0 : undefined,
                x: layout === "horizontal" ? undefined : yAxisWidth + 20,
              }}
              content={({ active, payload, label }) => {
                const cleanPayload: TooltipProps["payload"] = payload
                  ? payload.map((item: any) => ({
                    category: item.dataKey,
                    value: item.value,
                    index: item.payload[index],
                    color: categoryColors.get(
                      item.dataKey,
                    ) as AvailableChartColorsKeys,
                    type: item.type,
                    payload: item.payload,
                  }))
                  : [];
                const total = type === 'stacked' && (valueType === 'number' || valueType === 'duration') && payload ? payload.reduce((sum, item) => {
                  if (typeof item.value === 'number') {
                    return sum + item.value
                  }
                  return sum
                }, 0) : undefined

                return showTooltip &&
                  active &&
                  hoveredIndex != null &&
                  hoveredChartId === chartId ? (
                  <ChartTooltip
                    active={active}
                    payload={cleanPayload}
                    label={indexFormatter(label)}
                    valueFormatter={valueFormatter}
                    extraData={extraDataByIndexAxis[label]}
                    total={total}
                  />
                ) : null;
              }}
            />
            {showLegend ? (
              <RechartsLegend
                verticalAlign="top"
                height={legendHeight}
                content={({ payload }) =>
                  ChartLegend(
                    { payload },
                    categoryColors,
                    setLegendHeight,
                    undefined,
                    undefined,
                    enableLegendSlider,
                  )
                }
              />
            ) : null}
            {hoveredIndex != null && hoveredIndexType === indexType && hoveredChartId !== chartId && (
              <ReferenceLine
                x={layout === "horizontal" ? hoveredIndex : undefined}
                y={layout === "vertical" ? hoveredIndex : undefined}
                stroke="var(--shaper-reference-line-color)"
              />
            )}
            {categories.map((category) => (
              <Bar
                className={cx(
                  getColorClassName(
                    categoryColors.get(category) as AvailableChartColorsKeys,
                    "fill",
                  )
                )}
                key={category}
                name={category}
                type="linear"
                dataKey={category}
                stackId={stacked ? "stack" : undefined}
                isAnimationActive={false}
                fill=""
                shape={(props: any) =>
                  renderShape(props, undefined, undefined, layout)
                }
              />
            ))}
          </RechartsBarChart>
        </ResponsiveContainer>
      </div>
    );
  },
);

BarChart.displayName = "BarChart";

export { BarChart, type BarChartEventProps, type TooltipProps };
