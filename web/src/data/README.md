# Site data

Data files the site reads at build time.

## `ve-states.json`

Outline of each Venezuelan state, as SVG paths, for the map on the home page.

|             |                                                                                                                                                                                                              |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Source**  | Venezuela – Subnational Administrative Boundaries (COD-AB), OCHA, on [HDX](https://data.humdata.org/dataset/5b141d29-534f-4f01-a0bc-41e2f375d925). The same dataset as `data/geography/` in this repository. |
| **File**    | `ven_admin1.geojson` from `ven_admin_boundaries.geojson.zip`                                                                                                                                                 |
| **License** | Creative Commons Attribution for Intergovernmental Organisations (CC BY-IGO)                                                                                                                                 |
| **Codes**   | `code` is the OCHA P-code (`VE14`), the same code the API uses.                                                                                                                                              |

Shape of the file:

```json
{
  "viewBox": [0, 0, 1000, 849],
  "states": [
    { "code": "VE14", "name": "Mérida", "d": "M…Z", "label": [154, 273] }
  ]
}
```

`label` is a point inside the state's largest polygon, in `viewBox` units.

### How to regenerate it

The raw boundaries are several MB, so they are not committed. When OCHA
publishes new boundaries:

1. Download `ven_admin_boundaries.geojson.zip` from the HDX page above and
   unzip `ven_admin1.geojson` into `web/tmp/` (ignored by git).
2. From `web/`:

   ```bash
   pnpm map:simplify   # mapshaper: Mercator projection + 2% simplification
   pnpm map:build      # scripts/build-map.mjs: GeoJSON -> SVG paths
   ```

Small islands under 0.2% of a state's largest polygon are dropped to keep
the file small (about 50 KB).
