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

export interface DashboardProps {
  id: string;
  vars: VarsParamSchema;
  getJwt: () => Promise<string>;
  baseUrl?: string;
  onVarsChanged: (newVars: VarsParamSchema) => void;
  hash?: string;
  menuButton?: React.ReactNode;
  onError?: (error: Error) => void;
  data?: Result;
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
  data: initialData,
}: DashboardProps) {
  const [fetchedData, setFetchedData] = useState<Result | null>(null);
  const [error, setError] = useState<Error | null>();

  useEffect(() => {
    if (initialData) {
      return;
    }

    setError(null);
    fetchDashboard(id, vars, baseUrl, getJwt)
      .then(setFetchedData)
      .catch((err) => {
        if (err?.isRedirect) {
          onError?.(err);
          return;
        }
        setError(err);
      });
  }, [id, vars, baseUrl, hash, onError, getJwt, initialData]);

  const data = initialData ?? fetchedData;

  if (error) {
    return (
      <div className="min-h-[calc(100vh)] flex items-center justify-center text-xl">
        <div className="p-4 m-4 bg-red-200 rounded-md font-mono">
          <p>{error.message}</p>
        </div>
      </div>
    );
  }

  if (!data) {
    return (
      <div className="min-h-[calc(100vh)] flex items-center justify-center text-xl">
        <span>{translate("loading")}...</span>
      </div>
    );
  }

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

  return (
    <ChartHoverProvider>
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
                className={cx("sm:flex-grow flex items-center ml-1", {
                  "w-full sm:w-fit": section.title,
                })}
              >
                {sectionIndex === 0 ? (
                  <>
                    {menuButton}
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
              "sm:grid-cols-2": numQueriesInSection > 1,
              "lg:grid-cols-2":
                numQueriesInSection === 2 ||
                (numContentSections === 1 && numQueriesInSection === 4),
              "lg:grid-cols-3":
                numQueriesInSection > 4 ||
                numQueriesInSection === 3 ||
                (numQueriesInSection === 4 && numContentSections > 1),
              "xl:grid-cols-4":
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
                    "mr-4 mb-4 p-4 h-[calc(50vh-2.6rem)] min-h-[18rem]",
                    {
                      "h-[calc(65vh-4.7rem)] sm:h-[calc(100vh-4.7rem)]":
                        numQueriesInSection === 1,
                      "lg:h-[calc(100vh-4.7rem)]":
                        numContentSections === 1 && numQueriesInSection === 2,
                    },
                  )}
                >
                  {query.render.label ? (
                    <h2 className="text-md mb-4 text-center font-semibold font-display">
                      {query.render.label}
                    </h2>
                  ) : null}
                  <div
                    className={cx({
                      "h-[calc(100%-2rem)]": query.render.label,
                      "h-full": !query.render.label,
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
        <div className="text-center text-ctext2 dark:text-dtext2 leading-[calc(70vh)]">
          Nothing to show yet ...
        </div>
      ) : null}
    </ChartHoverProvider>
  );
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
      <div className="h-full py-1 px-3 flex items-center justify-center text-ctext2 dark:text-dtext2">
        {translate("No data available")}
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
      (json?.Error?.Msg ?? json?.Error ?? json?.message ?? json).toString(),
    );
  }
  return json;
};
