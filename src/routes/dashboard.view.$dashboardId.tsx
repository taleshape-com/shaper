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

	return (
		<div className="mx-2 mt-2 mb-10 sm:mx-2 sm:mt-2">
			<Helmet>
				<title>{data.title}</title>
				<meta name="description" content={data.title} />
			</Helmet>
			<div className="flex items-center mx-2 pb-8">
				<h1 className="text-lg text-slate-700 flex-grow">{data.title}</h1>
				{data.queries.filter(({ render }) => render.type === 'dropdown' || render.type === 'dropdownMulti').map(({ render, columns, rows }, index) => {
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
			</div>
			{data.queries.length === 0 ? (
				<div>No data to show...</div>
			) :
				<div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3">
					{data.queries.filter(({ render }) => render.type !== 'dropdown' && render.type !== 'dropdownMulti').map(({ render, columns, rows }, index) => {
						if (render.type === "linechart") {
							return (
								<DashboardLineChart
									key={index}
									label={render.label}
									headers={columns}
									data={rows}
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
								/>
							);
						}
						return (
							<DashboardTable
								key={index}
								label={render.label}
								headers={columns}
								data={rows}
							/>
						);
					})}
				</div>}
		</div>
	);
}
