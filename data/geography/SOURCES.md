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

## Convenciones

- **Municipios sin división parroquial.** Algunos municipios no se dividen en
  parroquias (el INE los marca como "no tiene parroquia"). Se representan con
  **una única parroquia con el mismo nombre que el municipio**, para que la
  jerarquía estado → municipio → parroquia esté siempre completa. Un municipio
  con exactamente una parroquia es, por definición, uno de estos casos.

## Correcciones

### 1. Parroquias de municipios sin división parroquial

La fuente nombra estas parroquias de forma inconsistente: con prefijos estadísticos
del INE (`Capital X`, `Urbana X`, `Sección Capital X`), con el nombre de la capital
del municipio o con nombres incompletos. Se renombran según la convención de arriba.
El código no cambia.

| Código             | Antes                             | Después                  |
| ------------------ | --------------------------------- | ------------------------ |
| VE1611 (municipio) | Santa Babara                      | Santa Bárbara            |
| VE031501           | Capital San José de Guanipa       | San José de Guanipa      |
| VE040201           | Urbana Biruaca                    | Biruaca                  |
| VE050101           | Capital Bolívar                   | Bolívar                  |
| VE050401           | Capital José Ángel Lamas          | José Ángel Lamas         |
| VE050601           | Capital José Rafael Revenga       | José Rafael Revenga      |
| VE051001           | Capital San Sebastián             | San Sebastián            |
| VE051401           | Capital Tovar                     | Tovar                    |
| VE070301           | Sección Capital el Callao         | El Callao                |
| VE071101           | Sección Capital Padre Pedro Chien | Padre Pedro Chien        |
| VE080701           | Urbana los Guayos                 | Los Guayos               |
| VE080801           | Urbana Miranda                    | Miranda                  |
| VE080901           | Urbana Montalbán                  | Montalbán                |
| VE081001           | Urbana Naguanagua                 | Naguanagua               |
| VE081201           | Urbana San Diego                  | San Diego                |
| VE081301           | Urbana San Joaquín                | San Joaquín              |
| VE090501           | El Pao                            | Pao de San Juan Bautista |
| VE150501           | Mamporal                          | Buroz                    |
| VE151301           | San Antonio de los Altos          | Los Salías               |
| VE151501           | Santa Lucia                       | Paz Castillo             |
| VE151701           | Guarenas                          | Plaza                    |
| VE161101           | Santa Barbará                     | Santa Bárbara            |
| VE170101           | Capital Antolín del Campo         | Antolín del Campo        |
| VE170201           | Capital Arismendi                 | Arismendi                |
| VE170801           | Capital Mariño                    | Mariño                   |
| VE180101           | Capital Agua Blanca               | Agua Blanca              |
| VE190601           | Capital Bolívar                   | Bolívar                  |
| VE191101           | Capital Mejía                     | Mejía                    |
| VE200101           | Capital Andrés Bello              | Andrés Bello             |
| VE200201           | Antonio Rómulo Acosta             | Antonio Rómulo Costa     |
| VE200601           | Capital Córdoba                   | Córdoba                  |
| VE200801           | San José de Bolívar               | Francisco de Miranda     |
| VE201001           | Capital Guasimos                  | Guasimos                 |
| VE201301           | Vargas                            | José María Vargas        |
| VE201801           | Capital Michelena                 | Michelena                |
| VE202101           | Delicias                          | Rafael Urdaneta          |
| VE202401           | Capital Seboruco                  | Seboruco                 |

Notas:

- `VE1611`: el municipio se corrige de `Santa Babara` a `Santa Bárbara` (error de tipeo).
- `VE200201`: el censo INE 2011 escribe "Acosta", pero el municipio lleva el nombre de
  Antonio Rómulo Costa (1872–1956) y el propio INE usa "Costa" en otras publicaciones.
- No se renombra `VE090901` (Tinaco → `General en Jefe José Laurencio Silva`):
  OCHA y otras fuentes coinciden en ese nombre.

### 2. Prefijos estadísticos del INE

La fuente antepone etiquetas de clasificación estadística del INE al nombre de
145 parroquias: `Capital`, `Capital Urbana`, `Sección Capital`, `Urbana` y
`No Urbana`. No forman parte del nombre y se eliminan con una regla mecánica:
quitar el prefijo y poner en mayúscula la primera letra.

| Antes             | Después     |
| ----------------- | ----------- |
| No Urbana Chuao   | Chuao       |
| Urbana los Guayos | Los Guayos  |
| Capital Guanare   | Guanare     |
| Urbanafraternidad | Fraternidad |

En los municipios con varias parroquias, la parroquia `Capital X` queda con el
nombre del municipio (`Capital Miranda` → `Miranda`). Es la convención que usa el
INE en sus publicaciones del censo 2011 (verificado en Táchira). Otras fuentes
usan el nombre del pueblo capital (ej. `Timotes`, `Lechería`); ver problemas conocidos.

## Problemas conocidos

- Algunos nombres no tienen tilde o tienen errores (ej. `Antimano`, `San Bernandino`).
- Faltan parroquias en al menos 5 municipios (ej. Maroa, Julio César Salas, Francisco Javier Pulgar);
  los huecos en la numeración de códigos lo delatan.
- Algunos municipios de Amazonas incluyen el prefijo `Autónomo` en el nombre.
- La fuente refleja la división vigente al 2021-02-23; cambios posteriores deben verificarse.
- En municipios con varias parroquias, la parroquia capital lleva el nombre del municipio
  (convención del INE). Algunas fuentes la nombran por el pueblo capital (ej. Miranda/Timotes
  en Mérida, Diego Bautista Urbaneja/Lechería en Anzoátegui); sin verificar contra leyes estadales.
- No incluye capitales de estados ni de municipios.
