# Shaper

**Open Source, SQL-driven Data Dashboards powered by DuckDB.**

> **Need a secure, compliant setup for sensitive data?** Maintain 100% data sovereignty while we handle the operations and infrastructure. [Get a managed, private Shaper instance with expert support.](https://taleshape.com/plans-and-pricing)

---

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

Render a GitHub-style calendar heatmap of date-based activity with `::HEATMAP`:

```sql
SELECT 'Activity'::LABEL;

SELECT
  date_trunc('day', created_at)::XAXIS,
  count(*)::HEATMAP
FROM events
GROUP BY ALL
ORDER BY ALL;
```

[
![Screenshot](https://taleshape.com/images/session_dashboard.png)
](https://taleshape.com/shaper/docs/)

Learn more:
https://taleshape.com/shaper/docs/


## Features

**Data Visualization**

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


## Managed Hosting and Expert Support

Shaper is 100% free and open source. Through **Taleshape**, we provide managed deployments and fractional data engineering for teams in regulated industries that need to move fast while maintaining strict data sovereignty:

- **Managed Private Cloud**: Dedicated, isolated instances in our cloud or your own infrastructure. We handle updates, security, and 24/7 monitoring.
- **Fractional Data Team**: Proactive support for integrations, custom dashboard development, and compliance readiness (HIPAA, GDPR, SOC2).

[**View Plans and Pricing**](https://taleshape.com/plans-and-pricing)


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
