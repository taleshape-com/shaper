# Shaper

**Open Source, SQL-driven Data Dashboards powered by DuckDB.**

Build analytics dashboards simply by writing SQL:

```sql
SELECT 'Sessions per Week'::LABEL;
SELECT
  date_trunc('week', created_at)::XAXIS,
  category::CATEGORY,
  count()::BARCHART_STACKED,
FROM dataset
GROUP BY ALL ORDER BY ALL;
```

[
![Screenshot](https://taleshape.com/_astro/session_dashboard.DjtFqCnO_Z15ug1D.webp)
](https://taleshape.com/shaper/docs/)

Learn more:
https://taleshape.com/shaper/docs/


## Features

**Business Intelligence**

- **Open Source** & Self-Hosted
- **SQL-First** and AI-Ready
- **Git-Based** Workflow
- Query across **Data Sources**

**Embedded Analytics**

- **White-Labeling** & custom styles
- **Row-level security** via JWT tokens
- Embed **Without IFrame** through JS & React SDKs

**Automated Reporting**

- Generate **PDF, PNG, CSV & Excel**
- Scheduled **Alerts & Reports**
- Sharable **Password-Protected Links**


## Quickstart

The quickest way to try out Shaper without installing anything is to run it via [Docker](https://www.docker.com/):
```sh
docker run --rm -it -p5454:5454 taleshape/shaper
```
Then open http://localhost:5454/new in your browser.

For more, checkout the [Getting Started Guide](https://taleshape.com/shaper/docs/getting-started/).

To run Shaper in production, see the [Deployment Guide](https://taleshape.com/shaper/docs/deploy-to-production/).


## Support and Managed Hosting

Shaper itself is completely free and open source.
But we offer managed hosting and proactive support.
Find out more:

[Plans and Pricing](https://taleshape.com/plans-and-pricing)


## Get in touch

Feel free to open an [issue](https://github.com/taleshape-com/shaper/issues) or start a [discussion](https://github.com/taleshape-com/shaper/discussions) if you have any questions or suggestions.

Also follow along on [BlueSky](https://bsky.app/profile/taleshape.bsky.social) or [LinkedIn](https://www.linkedin.com/company/taleshape/).

And subscribe to our [newsletter](https://taleshape.com/newsletter) to get updates about Shaper.


## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md)


## Release Notes

See [Github Releases](https://github.com/taleshape-com/shaper/releases)


## License and Copyright

Shaper is licensed under the [Mozilla Public License 2.0](https://github.com/taleshape-com/shaper/blob/main/LICENSE).

Copyright © 2024-2026 Taleshape OÜ
