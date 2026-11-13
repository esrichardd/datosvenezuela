# Fuentes: geografía

## Fuente principal

|                                |                                                                                     |
| ------------------------------ | ----------------------------------------------------------------------------------- |
| **Dataset**                    | Venezuela (Bolivarian Republic of) – Subnational Administrative Boundaries (COD-AB) |
| **Publicado por**              | OCHA Field Information Services Section (FISS), en Humanitarian Data Exchange (HDX) |
| **Origen de los datos**        | Instituto Nacional de Estadística (INE)                                             |
| **URL**                        | https://data.humdata.org/dataset/5b141d29-534f-4f01-a0bc-41e2f375d925               |
| **Archivo**                    | `ven_admin_boundaries.xlsx` (versión `v01`, `valid_on` 2021-02-23)                  |
| **Última modificación en HDX** | 2026-08-14                                                                          |
| **Descargado**                 | 2026-11-13                                                                          |
| **Licencia**                   | Creative Commons Attribution for Intergovernmental Organisations (CC BY-IGO)        |

## Archivos

| Archivo              | Registros | Columnas                             |
| -------------------- | --------- | ------------------------------------ |
| `states.csv`         | 25        | `code`, `name`, `type`, `iso_3166_2` |
| `municipalities.csv` | 335       | `code`, `state_code`, `name`         |
| `parishes.csv`       | 1.134     | `code`, `municipality_code`, `name`  |

Los códigos (`code`) son los P-codes del dataset de OCHA. Siguen la jerarquía
estado (`VE14`) → municipio (`VE1401`) → parroquia (`VE140101`).

## Transformaciones respecto a la fuente

1. **Dependencias Federales (`VE25`)**: se conserva como entidad en `states.csv`,
   pero se excluyen el municipio `VE2501` y la parroquia `VE250101`. La fuente los
   creó solo para completar la jerarquía; no existen como municipio ni parroquia reales.
2. **`type`**: agregado por este proyecto (`state`, `capital_district`, `federal_dependencies`).
3. **`iso_3166_2`**: agregado por este proyecto según el estándar ISO 3166-2:VE.
4. Nombres: copiados tal cual de la fuente. Las correcciones se registran abajo.

## Correcciones

Ninguna todavía.

## Problemas conocidos

- Algunos nombres no tienen tilde o tienen errores (ej. `Antimano`, `San Bernandino`).
- Algunos municipios de Amazonas incluyen el prefijo `Autónomo` en el nombre.
- La fuente refleja la división vigente al 2021-02-23; cambios posteriores deben verificarse.
- No incluye capitales de estados ni de municipios.
