# Shaper

**Open source, SQL-first data dashboards, reports and customer-facing analytics.**

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

To run Shaper in production, see the [Deployment Guide](https://taleshape.com/shaper/docs/deploy-to-production/). There you'll find information not only on how to deploy to production, but also how to do so securely and lock down your system.


## Managed Hosting & Expert Support

Shaper is 100% free and open source. All features are included — there is no paid edition.

Through [**Taleshape**](https://taleshape.com), we offer managed hosting and hands-on support for teams looking for help getting analytics into production:

- **Managed Hosting**: We deploy, maintain, and monitor Shaper in your infrastructure. We handle updates, security monitoring, and proactive patch delivery.
- **Hands-On Support**: Help with integrations, building dashboards, and meeting compliance requirements (HIPAA, GDPR, BSI C5, SOC 2).
- **Zero Direct Access**: We deliver all changes as Pull Requests to your repository. No SSH, no VPN, no backdoors. You review and approve every change.

[**Learn more about our services**](https://taleshape.com/plans-and-pricing)


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
