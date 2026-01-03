**Open Source, SQL-driven Data Dashboards powered by DuckDB.**

Learn more:

https://taleshape.com/shaper/docs/

---

Install Shaper via PyPI when using Shaper locally if you already have Python installed.

Python's package manager `pip` manages the version for you and handles downloading the correct binary for your system.

If you have `pipx` installed, you can run Shaper via `pipx` without explicitly installing it:
```bash
pipx run shaper-bin
```

Install Shaper globally to make the `shaper` binary available in your PATH:
```bash
pipx install shaper-bin
```

You can also use `pip` to install Shaper as a dependency in your project. This is useful to ensure everyone working on the project uses the same version of shaper:
```bash
pip install shaper-bin
```

To run Shaper in production, we recommend using the Docker image since it ensures a consistent environment.

Find more detailed installation and usage instructions in the documentation:

https://taleshape.com/shaper/docs/installing-shaper/


## License and Copyright

Shaper is licensed under the [Mozilla Public License 2.0](https://github.com/taleshape-com/shaper/blob/main/LICENSE).

Copyright © 2024-2026 Taleshape OÜ
