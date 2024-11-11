import { z } from "zod";
import { createFileRoute, Link, notFound } from "@tanstack/react-router";
import type { ErrorComponentProps } from "@tanstack/react-router";
import { Result } from "../lib/dashboard";
import DashboardLineChart from "../components/dashboard/DashboardLineChart";
import DashboardTable from "../components/dashboard/DashboardTable";
import { redirect } from "@tanstack/react-router";
import { Helmet } from "react-helmet";
import DashboardBarChart from "../components/dashboard/DashboardBarChart";
import DashboardDropdown from "../components/dashboard/DashboardDropdown";
import { useNavigate } from "@tanstack/react-router";
import DashboardDropdownMulti from "../components/dashboard/DashboardDropdownMulti";
import { cx } from "../lib/utils";

export const Route = createFileRoute("/dashboard/view/$dashboardId")({
  validateSearch: z.object({
    vars: z.record(z.union([z.string(), z.array(z.string())])).optional(),
  }),
  loaderDeps: ({ search: { vars } }) => ({
    vars,
  }),
  loader: async ({ params: { dashboardId }, deps: { vars } }) => {
    const urlVars = Object.entries(vars ?? {}).reduce((acc, [key, value]) => {
      if (Array.isArray(value)) {
        return [...acc, ...value.map((v) => [key, v])];
      }
      return [...acc, [key, value]];
    }, [] as string[][])
    const searchParams = new URLSearchParams(urlVars).toString();
    return fetch(`/api/dashboard/${dashboardId}?${searchParams}`)
      .then(async (response) => {
        if (response.status === 401) {
          throw redirect({
            to: "/login",
            search: {
              // Use the current location to power a redirect after login
              // (Do not use `router.state.resolvedLocation` as it can
              // potentially lag behind the actual current location)
              redirect: location.pathname + location.search + location.hash,
            },
          });
        }
        if (response.status === 404) {
          throw notFound();
        }
        if (response.status !== 200) {
          return response
            .json()
            .then(
              (data: {
                Error?: { Type: number; Msg?: string };
                message?: string;
              }) => {
                throw new Error(
                  (
                    data?.Error?.Msg ??
                    data?.Error ??
                    data?.message ??
                    data
                  ).toString(),
                );
              },
            );
        }
        return response.json();
      })
      .then((fetchedData: Result) => {
        return fetchedData;
      });
  },
  errorComponent: DashboardErrorComponent,
  notFoundComponent: () => {
    return (
      <div>
        <p>Dashboard not found</p>
        <Link to="/" className="underline">
          Go to homepage
        </Link>
      </div>
    );
  },
  component: DashboardViewComponent,
});

function DashboardErrorComponent({ error }: ErrorComponentProps) {
  return (
    <div className="p-4 m-4 bg-red-200 rounded-md">
      <p>{error.message}</p>
    </div>
  );
}

function DashboardViewComponent() {
  const { vars } = Route.useSearch();
  const data = Route.useLoaderData();
  const navigate = useNavigate({ from: "/dashboard/view/$dashboardId" });

  if (!data) {
    return <div>Loading...</div>;
  }
  const onDropdownChange = (newVars: Record<string, string | string[]>) => {
    navigate({
      search: (old: any) => ({
        ...old,
        vars: newVars,
      }),
    });
  }
  const sections: ['menu' | 'content', Result['queries']][] = [['menu', []]];
  data.queries.forEach((query) => {
    const lastSection = sections[sections.length - 1];
    if (query.render.type === 'dropdown' || query.render.type === 'dropdownMulti') {
      if (lastSection[0] === 'menu') {
        lastSection[1].push(query);
        return;
      }
      sections.push(['menu', [query]]);
      return;
    }
    if (lastSection[0] === 'content') {
      lastSection[1].push(query);
      return;
    }
    sections.push(['content', [query]]);
    return;
  });

  return (
    <div className="mx-2 mt-2 mb-10 sm:mx-2 sm:mt-2">
      <Helmet>
        <title>{data.title}</title>
        <meta name="description" content={data.title} />
      </Helmet>
      {sections.map(([sectionType, queries], index) => {
        if (sectionType === 'menu') {
          return <section key={index} className={cx(["flex items-center mx-2 pb-8", index !== 0 ? "pt-8 border-t" : ""])}>
            {index === 0 ? <h1 className="text-lg text-slate-700 flex-grow py-1">{data.title}</h1> : <div className="flex-grow"></div>}
            {queries.map(({ render, columns, rows }, index) => {
              if (render.type === "dropdown") {
                return (
                  <DashboardDropdown
                    key={index}
                    label={render.label}
                    headers={columns}
                    data={rows}
                    onChange={onDropdownChange}
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
                    onChange={onDropdownChange}
                  />
                );
              }
            })}
          </section>
        }
        return <section
          key={index}
          className={cx({
            ["grid grid-cols-1"]: true,
            ["lg:grid-cols-2"]: queries.length === 2,
            ["md:grid-cols-2 xl:grid-cols-3"]: queries.length >= 3,
            ["xl:grid-cols-4"]: queries.length >= 4,
          })}
        >
          {queries.map(({ render, columns, rows }, index) => {
            if (render.type === "linechart") {
              return (
                <DashboardLineChart
                  key={index}
                  label={render.label}
                  headers={columns}
                  data={rows}
                  sectionCount={queries.length}
                />
              );
            }
            if (render.type === "barchart") {
              return (
                <DashboardBarChart
                  key={index}
                  label={render.label}
                  headers={columns}
                  data={rows}
                  sectionCount={queries.length}
                />
              );
            }
            return (
              <DashboardTable
                key={index}
                label={render.label}
                headers={columns}
                data={rows}
                sectionCount={queries.length}
              />
            );
          })}
        </section>
      })}
      {sections.length === 1 ? (
        <div className="text-center text-slate-600 leading-[calc(70vh)]">
          Nothing to show yet ...
        </div>
      ) : null}
    </div>
  );
}
