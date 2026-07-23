# Application template icon sources

Application template icons are vendored into this directory so the marketplace does not depend on a third-party CDN at runtime.

Asset selection follows this order:

1. an official SVG or raster asset published by the application project;
2. a maintained brand glyph from [Simple Icons](https://simpleicons.org/), with the template's official repository used to verify the product identity;
3. the neutral `fallback.svg` placeholder when no trustworthy reusable asset is available.

Do not draw an approximate logo for an established third-party product. Keep the original aspect ratio, use SVG whenever available, and record the source when adding or replacing an asset.

## Simple Icons assets

The following glyphs are vendored from Simple Icons `16.7.0` under its [CC0 1.0 license](https://github.com/simple-icons/simple-icons/blob/16.7.0/LICENSE.md). The fill color is set to the corresponding brand color for marketplace display.

| Local asset | Simple Icons slug | Official project |
| --- | --- | --- |
| `adminer.svg` | `adminer` | <https://github.com/vrana/adminer> |
| `caddy.svg` | `caddy` | <https://github.com/caddyserver/caddy> |
| `clickhouse.svg` | `clickhouse` | <https://github.com/ClickHouse/ClickHouse> |
| `docker-registry.svg` | `docker` | <https://github.com/distribution/distribution> |
| `excalidraw.svg` | `excalidraw` | <https://github.com/excalidraw/excalidraw> |
| `gitea.svg` | `gitea` | <https://github.com/go-gitea/gitea> |
| `grafana.svg` | `grafana` | <https://github.com/grafana/grafana> |
| `mariadb.svg` | `mariadb` | <https://github.com/MariaDB/server> |
| `meilisearch.svg` | `meilisearch` | <https://github.com/meilisearch/meilisearch> |
| `mongodb.svg` | `mongodb` | <https://github.com/mongodb/mongo> |
| `nats.svg` | `natsdotio` | <https://github.com/nats-io/nats-server> |
| `prometheus.svg` | `prometheus` | <https://github.com/prometheus/prometheus> |
| `uptime-kuma.svg` | `uptimekuma` | <https://github.com/louislam/uptime-kuma> |
| `vaultwarden.svg` | `vaultwarden` | <https://github.com/dani-garcia/vaultwarden> |
| `verdaccio.svg` | `verdaccio` | <https://github.com/verdaccio/verdaccio> |
| `wordpress.svg` | `wordpress` | <https://github.com/WordPress/WordPress> |

Other SVG files in this directory predate this manifest. Replace them with an official project asset when one is available, and add the provenance here.
