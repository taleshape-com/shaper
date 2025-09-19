// SPDX-License-Identifier: MPL-2.0

import { ErrorBoundary } from "react-error-boundary";
import { Result } from "../../lib/types";
import { ChartHoverProvider } from "../providers/ChartHoverProvider";
import { cx, getSearchParamString, VarsParamSchema } from "../../lib/utils";
import DashboardDropdown from "./DashboardDropdown";
import DashboardDropdownMulti from "./DashboardDropdownMulti";
import DashboardButton from "./DashboardButton";
import DashboardDatePicker from "./DashboardDatePicker";
import DashboardDateRangePicker from "./DashboardDateRangePicker";
import { Card } from "../tremor/Card";
import { translate } from "../../lib/translate";
import DashboardLineChart from "./DashboardLineChart";
import DashboardBarChart from "./DashboardBarChart";
import DashboardValue from "./DashboardValue";
import DashboardTable from "./DashboardTable";
import { useEffect, useState, useRef, useCallback } from "react";
import { RiBarChartFill, RiLayoutFill, RiLoader3Fill } from "@remixicon/react";
import DashboardGauge from "./DashboardGauge";
import { ChartDownloadButton } from "../charts/ChartDownloadButton";

export interface DashboardProps {
  id?: string;
  vars: VarsParamSchema;
  getJwt: () => Promise<string>;
  baseUrl?: string;
  onVarsChanged: (newVars: VarsParamSchema) => void;
  hash?: string;
  menuButton?: React.ReactNode;
  onError?: (error: Error) => void;
  data?: Result;
  onDataChange?: (data: Result) => void;
  loading?: boolean;
}

const MIN_SHOW_LOADING = 300;

export function Dashboard({
  id,
  vars,
  getJwt,
  baseUrl = window.shaper.defaultBaseUrl,
  onVarsChanged,
  // We update this string when JWT variables change to trigger a re-fetch
  hash = "",
  menuButton,
  onError,
  data,
  onDataChange,
  loading,
}: DashboardProps) {
  const [fetchedData, setFetchedData] = useState<Result | undefined>(undefined);
  const [error, setError] = useState<Error | null>(null);
  const [isFetching, setIsFetching] = useState<boolean>(false);
  const errResetFn = useRef<(() => void) | undefined>(undefined)

  const actualData = data ?? fetchedData;

  // Add timeout ref to store the timeout ID
  const reloadTimeoutRef = useRef<NodeJS.Timeout>();
  // Track the current AbortController so we can cancel in-flight requests
  const fetchAbortRef = useRef<AbortController | null>(null);

  // Function to fetch dashboard data
  const fetchData = useCallback(async () => {
    if (!id) return;

    // Abort any in-flight request before starting a new one
    if (fetchAbortRef.current) {
      fetchAbortRef.current.abort();
    }
    const abortController = new AbortController();
    fetchAbortRef.current = abortController;

    // Clear previous reload timer when starting a new fetch
    if (reloadTimeoutRef.current) {
      clearTimeout(reloadTimeoutRef.current);
      reloadTimeoutRef.current = undefined;
    }

    setError(null);
    setIsFetching(true);
    const startTime = Date.now();

    try {
      const d = await fetchDashboard(id, vars, baseUrl, getJwt, abortController.signal);
      const duration = Date.now() - startTime;
      await new Promise<void>(resolve => {
        setTimeout(() => {
          resolve();
        }, Math.max(0, MIN_SHOW_LOADING - duration));
      });

      // Only apply the results if this request is still the latest
      if (fetchAbortRef.current === abortController) {
        setFetchedData(d);
        if (onDataChange) {
          onDataChange(d);
        }
        // Set up reload timeout if reloadAt is in the future
        if (d.reloadAt > Date.now()) {
          const timeout = Math.max(1000, d.reloadAt - Date.now());
          reloadTimeoutRef.current = setTimeout(fetchData, timeout);
        }
      }
    } catch (err: unknown) {
      // Swallow abort errors (they are expected when a new request starts)
      if ((err as any)?.name === 'AbortError') {
        return;
      }
      setError(err as Error);
      onError?.(err as Error);
    } finally {
      // Only clear fetching state if this is still the latest request
      if (fetchAbortRef.current === abortController) {
        setIsFetching(false);
      }
    }
  }, [id, vars, baseUrl, getJwt, onDataChange, onError]);

  // Initial fetch and cleanup
  useEffect(() => {
    fetchData();
    return () => {
      // Clear timeouts on cleanup
      if (reloadTimeoutRef.current) {
        clearTimeout(reloadTimeoutRef.current);
      }
      // Abort any in-flight request on unmount
      if (fetchAbortRef.current) {
        fetchAbortRef.current.abort();
      }
    };
  }, [fetchData, hash]);

  useEffect(() => {
    if (errResetFn.current) {
      errResetFn.current();
      errResetFn.current = undefined;
    }
  }, [loading]);

  const ErrorDisplay = function({ error, resetErrorBoundary }: { error: Error, resetErrorBoundary?: () => void }) {
    errResetFn.current = resetErrorBoundary;
    return (
      <div className="antialiased text-ctext dark:text-dtext">
        {menuButton}
        <div>
          <div className="p-4 z-50 flex justify-center items-center">
            <div className="p-4 bg-red-100 text-red-700 h-fit rounded">
              {error.message}
            </div>
          </div>
        </div>
      </div>
    )
  }

  if (error) {
    return <ErrorDisplay error={error} />
  }

  return actualData ? (
    <ErrorBoundary fallbackRender={ErrorDisplay}>
      <DataView
        data={actualData}
        onVarsChanged={onVarsChanged}
        menuButton={menuButton}
        vars={vars}
        baseUrl={baseUrl}
        getJwt={getJwt}
        loading={loading || isFetching}
      />
    </ErrorBoundary>
  ) : (
    <ChartHoverProvider>
      <div className="w-full flex justify-center items-center flex-grow">
        <RiLoader3Fill className="size-7 fill-ctext dark:fill-dtext animate-spin" />
      </div>
    </ChartHoverProvider>
  );
}

const DataView = ({
  data,
  onVarsChanged,
  menuButton,
  vars,
  baseUrl,
  getJwt,
  loading,
}: (Pick<DashboardProps, 'onVarsChanged' | 'menuButton' | 'vars' | 'baseUrl' | 'getJwt'> & Required<Pick<DashboardProps, 'data'>>) & { loading: boolean }) => {
  const firstIsHeader = !(data.sections.length === 0 || data.sections[0].type !== "header");
  const sections: Result["sections"] = firstIsHeader
    ? data.sections
    : [
      {
        type: "header",
        queries: [],
      },
      ...data.sections,
    ];

  const numContentSections = sections.filter(
    (section) => section.type === "content",
  ).length;

  return (<ChartHoverProvider>
    {sections.map((section, sectionIndex) => {
      if (section.type === "header") {
        const queries = section.queries.filter(
          (query) => query.rows.length > 0,
        );
        return (
          <section
            key={sectionIndex}
            className={cx("flex flex-wrap items-center ml-2 mr-4", {
              "mt-3 mb-3": section.queries.length > 0 || section.title,
              "mt-4": section.title && sectionIndex !== 0,
              "my-2": section.queries.length === 0 && !section.title && sectionIndex === 0,
            })}
          >
            <div
              className={cx("@sm:flex-grow flex items-center ml-1", {
                "w-full @sm:w-fit": section.title,
              })}
            >
              {sectionIndex === 0 ? (
                <>
                  {menuButton}
                  {section.title ? (
                    <h1 className="text-2xl text-left ml-1 py-1 mt-0.5 font-semibold">
                      {section.title}
                    </h1>
                  ) : null}
                </>
              ) : section.title ? (
                <h2 className="text-xl text-left ml-1 mt-0.5 font-semibold">
                  {section.title}
                </h2>
              ) : null}
            </div>
            {queries.map(({ render, columns, rows }, index) => {
              if (render.type === "dropdown") {
                return (
                  <DashboardDropdown
                    key={index}
                    label={render.label}
                    headers={columns}
                    data={rows}
                    vars={vars}
                    onChange={onVarsChanged}
                  />
                );
              }
              if (render.type === "dropdownMulti") {
                return (
                  <DashboardDropdownMulti
                    key={index}
                    label={render.label}
                    headers={columns}
                    data={rows}
                    vars={vars}
                    onChange={onVarsChanged}
                  />
                );
              }
              if (render.type === "button") {
                return (
                  <DashboardButton
                    key={index}
                    label={render.label}
                    headers={columns}
                    data={rows}
                    baseUrl={baseUrl}
                    getJwt={getJwt}
                  />
                );
              }
              if (render.type === "datepicker") {
                return (
                  <DashboardDatePicker
                    key={index}
                    label={render.label}
                    headers={columns}
                    data={rows}
                    vars={vars}
                    onChange={onVarsChanged}
                  />
                );
              }
              if (render.type === "daterangePicker") {
                return (
                  <DashboardDateRangePicker
                    key={index}
                    label={render.label}
                    headers={columns}
                    data={rows}
                    vars={vars}
                    onChange={onVarsChanged}
                  />
                );
              }
            })}
          </section>
        );
      }

      const numQueriesInSection = section.queries.length;
      return (
        <section
          key={sectionIndex}
          className={cx("grid grid-cols-1 ml-4", {
            "@sm:grid-cols-2 print:grid-cols-2": numQueriesInSection > 1,
            "@lg:grid-cols-2 print:grid-cols-2":
              numQueriesInSection === 2 ||
              (numContentSections === 1 && numQueriesInSection === 4),
            "@lg:grid-cols-3 print:grid-cols-3":
              numQueriesInSection > 4 ||
              numQueriesInSection === 3 ||
              (numQueriesInSection === 4 && numContentSections > 1),
            "@xl:grid-cols-4":
              (numQueriesInSection === 4 && numContentSections > 1) ||
              numQueriesInSection === 7 ||
              numQueriesInSection === 8 ||
              numQueriesInSection > 9,
            "@4xl:grid-cols-5":
              (numQueriesInSection === 5 && numContentSections > 1) ||
              numQueriesInSection >= 9,
          })}
        >
          {section.queries.map((query, queryIndex) => {
            if (query.render.type === "placeholder") {
              return <div key={queryIndex}></div>;
            }
            const isChartQuery = query.render.type === 'linechart' || query.render.type === 'gauge' || query.render.type.startsWith('barchart');
            return (
              <Card
                key={queryIndex}
                className={cx(
                  "mr-4 mb-4 bg-cbgs dark:bg-dbgs border-none shadow-sm flex flex-col group",
                  {
                    "min-h-[320px] h-[calc(50dvh-3.15rem)] print:h-[320px]": section.queries.some(q => q.render.type !== "value") || numContentSections <= 2,
                    "h-[calc(50dvh-1.6rem)] print:h-[320px]": !firstIsHeader && numContentSections === 1,
                    "h-[calc(100cqh-5.3rem)]": numContentSections === 1 && numQueriesInSection === 1 && firstIsHeader,
                    "h-[calc(100cqh-2.2rem)] ": numContentSections === 1 && numQueriesInSection === 1 && !firstIsHeader,
                    "min-h-max h-fit print:min-h-max print:h-fit": section.queries.length === 1 && (query.render.type === "table" || (query.render.type === "value" && numContentSections > 2)),
                    "break-inside-avoid": query.render.type !== "table",
                  },
                )}
              >
                {isChartQuery && (
                  <ChartDownloadButton
                    chartId={`${sectionIndex}-${queryIndex}`}
                    label={query.render.label}
                    className="absolute top-2 right-2 z-40"
                  />
                )}
                {query.render.label && !isChartQuery ? (
                  <h2 className="text-md pt-4 mx-4 text-center font-semibold font-display">
                    {query.render.label}
                  </h2>
                ) : null}
                <div
                  className={cx("flex-1 relative overflow-auto", { "m-4": !isChartQuery })}
                >
                  {renderContent(
                    query,
                    sectionIndex,
                    queryIndex,
                    data.minTimeValue,
                    data.maxTimeValue,
                    numQueriesInSection,
                  )}
                </div>
              </Card>
            );
          })}
        </section>
      );
    })}
    {
      numContentSections === 0 ? (
        <div className="mt-32 flex flex-col items-center justify-center text-ctext2 dark:text-dtext2">
          <RiLayoutFill
            className="mx-auto size-9"
            aria-hidden={true}
          />
          <p className="mt-3 font-medium">
            {translate("Nothing to show yet")}
          </p>
        </div>
      ) : null
    }
    {loading && (
      <div className="sticky bottom-0 h-0 z-50 pointer-events-none w-full relative">
        <div className="p-1 bg-cbgs dark:bg-dbgs rounded-md shadow-md absolute right-2 bottom-2">
          <RiLoader3Fill className="size-7 fill-ctext dark:fill-dtext animate-spin" />
        </div>
      </div>
    )}
  </ChartHoverProvider >)
}

const renderContent = (
  query: Result["sections"][0]["queries"][0],
  sectionIndex: number,
  queryIndex: number,
  minTimeValue: number,
  maxTimeValue: number,
  numQueriesInSection: number,
) => {
  if (query.rows.length === 0) {
    return (
      <div className="h-full py-1 px-3 mb-1 flex items-center justify-center text-ctext2 dark:text-ctext2">
        <div className="text-center">
          <RiBarChartFill
            className="mx-auto size-9"
            aria-hidden={true}
          />
          <p className="mt-2 font-medium">
            {translate("No data")}
          </p>
        </div>
      </div>
    );
  }
  if (query.render.type === "linechart") {
    return (
      <DashboardLineChart
        chartId={`${sectionIndex}-${queryIndex}`}
        label={query.render.label}
        headers={query.columns}
        data={query.rows}
        minTimeValue={minTimeValue}
        maxTimeValue={maxTimeValue}
        markLines={query.render.markLines}
      />
    );
  }
  if (query.render.type === "gauge") {
    return (
      <DashboardGauge
        chartId={`${sectionIndex}-${queryIndex}`}
        headers={query.columns}
        data={query.rows}
        gaugeCategories={query.render.gaugeCategories}
        label={query.render.label}
      />
    );
  }
  if (
    query.render.type === "barchartHorizontal" ||
    query.render.type === "barchartHorizontalStacked" ||
    query.render.type === "barchartVertical" ||
    query.render.type === "barchartVerticalStacked"
  ) {
    return (
      <DashboardBarChart
        chartId={`${sectionIndex}-${queryIndex}`}
        label={query.render.label}
        stacked={
          query.render.type === "barchartHorizontalStacked" ||
          query.render.type === "barchartVerticalStacked"
        }
        vertical={
          query.render.type === "barchartVertical" ||
          query.render.type === "barchartVerticalStacked"
        }
        headers={query.columns}
        data={query.rows}
        minTimeValue={minTimeValue}
        maxTimeValue={maxTimeValue}
        markLines={query.render.markLines}
      />
    );
  }
  if (query.render.type === "value") {
    return <DashboardValue headers={query.columns} data={query.rows} yScroll={numQueriesInSection > 1} />;
  }
  return <DashboardTable headers={query.columns} data={query.rows} />;
};

const fetchDashboard = async (
  id: string,
  vars: VarsParamSchema,
  baseUrl: string,
  getJwt: () => Promise<string>,
  signal?: AbortSignal,
): Promise<Result> => {
  const jwt = await getJwt();
  const searchParams = getSearchParamString(vars);
  const res = await fetch(`${baseUrl}api/dashboards/${id}?${searchParams}`, {
    headers: {
      "Content-Type": "application/json",
      Authorization: jwt,
    },
    signal,
  });
  const json = await res.json();
  if (res.status !== 200) {
    throw new Error(
      (json?.error ?? json?.Error?.Msg ?? json?.Error ?? json?.message ?? json).toString(),
    );
  }
  return json;
};
