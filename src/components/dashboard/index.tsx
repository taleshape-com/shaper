import { Result } from "../../lib/dashboard";
import { ChartHoverProvider } from "../ChartHoverProvider";
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
import { useEffect, useState } from "react";
import { RiBarChartFill, RiLayoutFill, RiLoader3Fill } from "@remixicon/react";

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

export function Dashboard({
  id,
  vars,
  getJwt,
  baseUrl = "",
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
  const [fetching, setFetching] = useState<boolean>(false);
  data = data ?? fetchedData;

  useEffect(() => {
    if (!id) {
      // When controlling the data from outside, we don't need to fetch
      return;
    }
    setError(null);
    setFetching(true);
    fetchDashboard(id, vars, baseUrl, getJwt)
      .then((d) => {
        setFetchedData(d);
        setFetching(false);
        if (onDataChange) {
          onDataChange(d);
        }
      })
      .catch((err) => {
        setError(err);
        setFetching(false);
        onError?.(err);
      });
  }, [id, vars, baseUrl, hash, onError, getJwt, onDataChange]);

  if (error) {
    return (
      <div>
        {menuButton && (
          <div className="mt-3 ml-3">{menuButton}</div>
        )}
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

  return (
    <div className="@container relative">
      {data ? (
        <DataView
          data={data}
          onVarsChanged={onVarsChanged}
          menuButton={menuButton}
          vars={vars}
          baseUrl={baseUrl}
          getJwt={getJwt}
          loading={loading || fetching}
        />
      ) : (
        <div className="absolute w-full h-[calc(100vh)] flex justify-center items-center">
          <RiLoader3Fill className="size-7 fill-ctext dark:fill-ctext animate-spin" />
        </div>
      )}
    </div >
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
  const sections: Result["sections"] =
    data.sections.length === 0
      ? [
        {
          type: "header",
          queries: [],
        },
      ]
      : data.sections[0].type !== "header"
        ? [
          {
            type: "header",
            queries: [],
          },
          ...data.sections,
        ]
        : data.sections;

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
              "mb-2 mt-1":
                section.queries.length > 0 ||
                section.title ||
                sectionIndex === 0,
              "pt-4": section.title && sectionIndex !== 0,
            })}
          >
            <div
              className={cx("@sm:flex-grow flex items-center ml-1", {
                "w-full @sm:w-fit": section.title,
              })}
            >
              {sectionIndex === 0 ? (
                <>
                  {menuButton && (
                    <div className="mt-2">{menuButton}</div>
                  )}
                  {section.title ? (
                    <h1 className="text-2xl text-left ml-1 mt-0.5">
                      {section.title}
                    </h1>
                  ) : null}
                </>
              ) : section.title ? (
                <h2 className="text-xl text-left ml-1 mt-0.5">
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
            "@sm:grid-cols-2": numQueriesInSection > 1,
            "@lg:grid-cols-2":
              numQueriesInSection === 2 ||
              (numContentSections === 1 && numQueriesInSection === 4),
            "@lg:grid-cols-3":
              numQueriesInSection > 4 ||
              numQueriesInSection === 3 ||
              (numQueriesInSection === 4 && numContentSections > 1),
            "@xl:grid-cols-4":
              (numQueriesInSection === 4 && numContentSections > 1) ||
              numQueriesInSection === 7 ||
              numQueriesInSection === 8 ||
              numQueriesInSection > 9,
          })}
        >
          {section.queries.map((query, queryIndex) => {
            if (query.render.type === "placeholder") {
              return <div key={queryIndex}></div>;
            }
            return (
              <Card
                key={queryIndex}
                className={cx(
                  "mr-4 mb-4 h-[calc(50vh-2.6rem)] min-h-[18rem] bg-cbgl dark:bg-dbgl border-none shadow-sm",
                  {
                    "h-[calc(65vh-4.7rem)] @sm:h-[calc(100vh-4.7rem)]":
                      numQueriesInSection === 1,
                    "@lg:h-[calc(100vh-4.7rem)]":
                      numContentSections === 1 && numQueriesInSection === 2,
                  },
                )}
              >
                {loading && (
                  <div className="absolute w-full h-full p-4 z-50 backdrop-blur flex justify-center items-center rounded-md">
                    <RiLoader3Fill className="size-7 fill-ctext dark:fill-ctext animate-spin" />
                  </div>
                )}
                {query.render.label ? (
                  <h2 className="text-md mb-4 mt-4 text-center font-semibold font-display">
                    {query.render.label}
                  </h2>
                ) : null}
                <div
                  className={cx("pb-5 mx-4", {
                    "h-[calc(100%-3rem)]": query.render.label,
                    "h-full pt-4": !query.render.label,
                  })}
                >
                  {renderContent(
                    query,
                    sectionIndex,
                    queryIndex,
                    data.minTimeValue,
                    data.maxTimeValue,
                  )}
                </div>
              </Card>
            );
          })}
        </section>
      );
    })}
    {numContentSections === 0 ? (
      <div className="text-center text-ctext2 dark:text-dtext2">
        <div className="mt-4 flex min-h-[calc(75vh)] items-center justify-center">
          <div className="text-center">
            <RiLayoutFill
              className="mx-auto size-9"
              aria-hidden={true}
            />
            <p className="mt-3 font-medium">
              {translate("Nothing to show yet")}
            </p>
          </div>
        </div>
      </div>
    ) : null}
  </ChartHoverProvider>)
}

const renderContent = (
  query: Result["sections"][0]["queries"][0],
  sectionIndex: number,
  queryIndex: number,
  minTimeValue: number,
  maxTimeValue: number,
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
        headers={query.columns}
        data={query.rows}
        minTimeValue={minTimeValue}
        maxTimeValue={maxTimeValue}
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
      />
    );
  }
  if (query.render.type === "value") {
    return <DashboardValue headers={query.columns} data={query.rows} />;
  }
  return <DashboardTable headers={query.columns} data={query.rows} />;
};

const fetchDashboard = async (
  id: string,
  vars: VarsParamSchema,
  baseUrl: string,
  getJwt: () => Promise<string>,
): Promise<Result> => {
  const jwt = await getJwt();
  const searchParams = getSearchParamString(vars);
  const res = await fetch(`${baseUrl}/api/dashboards/${id}?${searchParams}`, {
    headers: {
      "Content-Type": "application/json",
      Authorization: jwt,
    },
  });
  const json = await res.json();
  if (res.status !== 200) {
    throw new Error(
      (json?.error ?? json?.Error?.Msg ?? json?.Error ?? json?.message ?? json).toString(),
    );
  }
  return json;
};
