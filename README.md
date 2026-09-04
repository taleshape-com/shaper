# Shaper

**Open source, SQL-first data dashboards, reports and customer-facing analytics.**

> **Want us to run it for you?** We offer managed hosting where your data stays in your infrastructure. [See plans and pricing.](https://taleshape.com/plans-and-pricing)

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

To run Shaper in production, see the [Deployment Guide](https://taleshape.com/shaper/docs/deploy-to-production/).


## Managed Hosting and Expert Support

Shaper is 100% free and open source. Through **Taleshape**, we offer managed hosting and hands-on support for teams who need help getting analytics into production:

- **Managed Hosting**: We run Shaper for you—in our cloud or your infrastructure. We handle updates, security, and monitoring.
- **Hands-On Support**: Help with integrations, building dashboards, and meeting compliance requirements.

[**View Plans and Pricing**](https://taleshape.com/plans-and-pricing)


## Shared Responsibility: Open Source vs. Managed Service

Security and compliance follow a shared-responsibility model depending on whether you self-host Shaper or use Taleshape managed hosting:

> **If you self-host Shaper**, you own your own compliance and security posture.  
> **If you use Taleshape managed hosting**, we take on the operational, security, and compliance heavy lifting while your data remains in your infrastructure.

| Responsibility Area | Self-Hosted (Open Source) | Taleshape Managed Hosting |
| :--- | :--- | :--- |
| **Infrastructure & Uptime** | You provision, configure, and monitor your servers | Taleshape manages infrastructure, scaling, and 24/7 uptime |
| **Security Patches & Updates** | You track releases, test, and apply updates | Taleshape applies zero-downtime updates and security patches |
| **Platform Hardening & TLS** | You configure firewalls, SSL/TLS, and network boundaries | Taleshape enforces secure defaults, encryption, and hardening |
| **Compliance & Audits** | You are responsible for your own audits and certifications (SOC 2, ISO 27001, HIPAA, GDPR) | Taleshape provides audit support, compliance documentation, and platform-level controls |
| **Data Ownership & Location** | Stored entirely within your infrastructure | Stored entirely within your infrastructure (we don't store your analytical data) |
| **Access Control & Permissions**| You manage users, API keys, and dashboard permissions | You manage users, API keys, and dashboard permissions |
| **Dashboards & SQL Queries** | You author and maintain queries and reports | You author queries, with optional expert support from Taleshape |

### What We Take On (Taleshape Managed)

- **Infrastructure & Maintenance**: Automated provisioning, continuous monitoring, and zero-downtime version upgrades.
- **Security & Vulnerability Management**: Rapid patching, TLS/SSL configuration, network isolation, and secrets protection.
- **Compliance & Audit Readiness**: Support with compliance documentation, security questionnaires, and audit readiness for frameworks like SOC 2, HIPAA, and GDPR.
- **Operational Reliability**: Automated health checks, backup management, and failover support.

### What You Own (Self-Hosted)

- **Host & Environment Security**: Hardening the host OS, managing container environments, network firewalls, and ingress controllers.
- **Maintenance & Patching**: Tracking releases, testing upgrades, and rolling out security fixes promptly (see [SECURITY.md](./SECURITY.md)).
- **Compliance & Governance**: Implementing and certifying all technical, administrative, and physical controls required for your organization's compliance standards.
- **Availability & Disaster Recovery**: Monitoring application health, managing backups, and planning disaster recovery.


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
