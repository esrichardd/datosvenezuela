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

| Archivo              | Registros | Columnas                                        |
| -------------------- | --------- | ----------------------------------------------- |
| `states.csv`         | 25        | `code`, `name`, `type`, `iso_3166_2`, `capital` |
| `municipalities.csv` | 335       | `code`, `state_code`, `name`                    |
| `parishes.csv`       | 1.134     | `code`, `municipality_code`, `name`             |

Los códigos (`code`) son los P-codes del dataset de OCHA. Siguen la jerarquía
estado (`VE14`) → municipio (`VE1401`) → parroquia (`VE140101`).

## Transformaciones respecto a la fuente

1. **Dependencias Federales (`VE25`)**: se conserva como entidad en `states.csv`,
   pero se excluyen el municipio `VE2501` y la parroquia `VE250101`. La fuente los
   creó solo para completar la jerarquía; no existen como municipio ni parroquia reales.
2. **`type`**: agregado por este proyecto (`state`, `capital_district`, `federal_dependencies`).
3. **`iso_3166_2`**: agregado por este proyecto según el estándar ISO 3166-2:VE.
4. **`capital`**: agregado por este proyecto. Nombre de la ciudad capital de cada
   entidad, contrastado con [marydn/venezuela-sql](https://github.com/marydn/venezuela-sql)
   (corrigiendo su escritura: `San Cristóbal`, `San Juan de los Morros`). Se usa la
   forma de uso común `Coro` (nombre histórico: Santa Ana de Coro). Las Dependencias
   Federales no tienen capital: el campo queda vacío.
5. Nombres: copiados tal cual de la fuente. Las correcciones se registran abajo.

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

### 3. Tildes y mayúsculas

La fuente omite tildes en muchos nombres. Se agregan solo cuando hay evidencia:
el censo INE 2011 (Táchira y Zulia), el dataset comunitario
[marydn/venezuela-sql](https://github.com/marydn/venezuela-sql), o la misma palabra
escrita con tilde en otros nombres del dataset ("consistencia interna").
No se aplican tildes que se sabe erróneas en esas fuentes (ej. `Salóm`, `Mará`, `Fé`).

| Código   | Antes                                | Después                              | Evidencia            |
| -------- | ------------------------------------ | ------------------------------------ | -------------------- |
| VE0207   | Autónomo Rio Negro                   | Autónomo Río Negro                   | consistencia interna |
| VE0314   | Piritu                               | Píritu                               | marydn               |
| VE1118   | Piritu                               | Píritu                               | marydn               |
| VE1122   | Tocopero                             | Tocópero                             | marydn               |
| VE1201   | Camaguan                             | Camaguán                             | marydn               |
| VE1305   | Moran                                | Morán                                | marydn               |
| VE1410   | Julio Cesar Sala                     | Julio César Sala                     | consistencia interna |
| VE1810   | San Genaro de Boconoito              | San Genaro de Boconoíto              | marydn               |
| VE1814   | Turen                                | Turén                                | marydn               |
| VE2010   | Guasimos                             | Guásimos                             | INE, marydn          |
| VE2102   | Bocono                               | Boconó                               | INE, marydn          |
| VE2112   | Motatan                              | Motatán                              | marydn               |
| VE2113   | Pampan                               | Pampán                               | marydn               |
| VE2305   | Colon                                | Colón                                | INE, marydn          |
| VE2311   | Machiques de Perija                  | Machiques de Perijá                  | INE, marydn          |
| VE2316   | Rosario de Perija                    | Rosario de Perijá                    | INE, marydn          |
| VE010102 | Antimano                             | Antímano                             | marydn               |
| VE020301 | Fernando Giron Tovar                 | Fernando Girón Tovar                 | marydn               |
| VE031401 | Piritu                               | Píritu                               | marydn               |
| VE032002 | Tomas Alfaro Calatrava               | Tomás Alfaro Calatrava               | consistencia interna |
| VE050302 | Choroni                              | Choroní                              | marydn               |
| VE050504 | Pao de Zarate                        | Pao de Zárate                        | marydn               |
| VE050702 | San Martin de Porres                 | San Martín de Porres                 | marydn               |
| VE050902 | Guiripa                              | Güiripa                              | marydn               |
| VE051104 | Samán de Guere                       | Samán de Güere                       | marydn               |
| VE060405 | Santa Lucia                          | Santa Lucía                          | INE, marydn          |
| VE070402 | Ikabaru                              | Ikabarú                              | marydn               |
| VE080201 | Guigue                               | Güigüe                               | marydn               |
| VE090102 | Juan de Mata Suarez                  | Juan de Mata Suárez                  | marydn               |
| VE090901 | General En Jefe José Laurencio Silva | General en Jefe José Laurencio Silva | marydn               |
| VE110301 | Capatarida                           | Capatárida                           | marydn               |
| VE110303 | Borojo                               | Borojó                               | marydn               |
| VE110306 | Zazarida                             | Zazárida                             | marydn               |
| VE110902 | Adicora                              | Adícora                              | marydn               |
| VE110909 | El Vinculo                           | El Vínculo                           | marydn               |
| VE111005 | Maparari                             | Mapararí                             | marydn               |
| VE111406 | Rio Seco                             | Río Seco                             | marydn               |
| VE111801 | Piritu                               | Píritu                               | marydn               |
| VE112201 | Tocopero                             | Tocópero                             | marydn               |
| VE112505 | Zazarida                             | Zazárida                             | marydn               |
| VE120101 | Camaguan                             | Camaguán                             | marydn               |
| VE130101 | Pio Tamayo                           | Pío Tamayo                           | marydn               |
| VE130103 | Yacambu                              | Yacambú                              | marydn               |
| VE130201 | Freitez                              | Fréitez                              | marydn               |
| VE130508 | Moran                                | Morán                                | marydn               |
| VE130702 | Buria                                | Buría                                | marydn               |
| VE130806 | Chiquinquira                         | Chiquinquirá                         | INE, marydn          |
| VE140502 | Capuri                               | Capurí                               | marydn               |
| VE140503 | Chacanta                             | Chacantá                             | marydn               |
| VE140507 | Mucuchachi                           | Mucuchachí                           | marydn               |
| VE140605 | Jaji                                 | Jají                                 | marydn               |
| VE140903 | Rio Negro                            | Río Negro                            | INE, marydn          |
| VE141203 | Caracciolo Parra Perez               | Caracciolo Parra Pérez               | INE, marydn          |
| VE141208 | Juan Rodríguez Suarez                | Juan Rodríguez Suárez                | marydn               |
| VE141704 | Mucuruba                             | Mucurubá                             | marydn               |
| VE142002 | Chiguara                             | Chiguará                             | marydn               |
| VE142003 | Estanquez                            | Estánquez                            | marydn               |
| VE150102 | Araguita                             | Aragüita                             | marydn               |
| VE151007 | Tacata                               | Tácata                               | marydn               |
| VE151401 | Rio Chico                            | Río Chico                            | marydn               |
| VE151601 | Cupira                               | Cúpira                               | marydn               |
| VE151902 | Caucaguita                           | Caucagüita                           | marydn               |
| VE152001 | Cua                                  | Cúa                                  | marydn               |
| VE152002 | Nueva Cua                            | Nueva Cúa                            | marydn               |
| VE170702 | Adrian                               | Adrián                               | marydn               |
| VE180202 | Rio Acarigua                         | Río Acarigua                         | marydn               |
| VE181001 | San Genaro de Boconoito              | San Genaro de Boconoíto              | marydn               |
| VE190301 | Rio Caribe                           | Río Caribe                           | marydn               |
| VE190404 | Guaraunos                            | Guaraúnos                            | marydn               |
| VE191501 | Guiria                               | Güiria                               | marydn               |
| VE191503 | Cristóbal Colon                      | Cristóbal Colón                      | marydn               |
| VE200303 | San Pedro del Rio                    | San Pedro del Río                    | INE, marydn          |
| VE200502 | Amenodoro Rangel Lamus               | Amenodoro Rangel Lamús               | INE                  |
| VE201001 | Guasimos                             | Guásimos                             | INE, marydn          |
| VE201402 | La Petrolea                          | La Petrólea                          | INE, marydn          |
| VE201403 | Quinimari                            | Quinimarí                            | INE, marydn          |
| VE201404 | Bramon                               | Bramón                               | INE, marydn          |
| VE210201 | Bocono                               | Boconó                               | INE, marydn          |
| VE210302 | Cheregue                             | Cheregüé                             | marydn               |
| VE210401 | Chejende                             | Chejendé                             | marydn               |
| VE210402 | Arnoldo Gabaldon                     | Arnoldo Gabaldón                     | marydn               |
| VE210802 | Arnoldo Gabaldon                     | Arnoldo Gabaldón                     | marydn               |
| VE211103 | Santa María del Horcon               | Santa María del Horcón               | marydn               |
| VE211201 | Motatan                              | Motatán                              | marydn               |
| VE211301 | Pampan                               | Pampán                               | marydn               |
| VE211403 | Pampanito Ii                         | Pampanito II                         | marydn               |
| VE211604 | José Leonardo Suarez                 | José Leonardo Suárez                 | marydn               |
| VE211802 | Chiquinquira                         | Chiquinquirá                         | INE, marydn          |
| VE211902 | Cabimbu                              | Cabimbú                              | marydn               |
| VE211903 | Jajo                                 | Jajó                                 | marydn               |
| VE230402 | Udon Perez                           | Udón Pérez                           | INE, marydn          |
| VE230505 | Urribarri                            | Urribarrí                            | INE, marydn          |
| VE230702 | José Ramón Yepez                     | José Ramón Yépez                     | marydn               |
| VE230802 | Bari                                 | Barí                                 | INE, marydn          |
| VE230903 | Chiquinquira                         | Chiquinquirá                         | INE, marydn          |
| VE231103 | Rio Negro                            | Río Negro                            | INE, marydn          |
| VE231104 | San José de Perija                   | San José de Perijá                   | INE, marydn          |
| VE231304 | Caracciolo Parra Perez               | Caracciolo Parra Pérez               | INE, marydn          |
| VE231308 | Chiquinquira                         | Chiquinquirá                         | INE, marydn          |
| VE231316 | Santa Lucia                          | Santa Lucía                          | INE, marydn          |
| VE231403 | Faria                                | Faría                                | INE, marydn          |
| VE231803 | José Cenovio Urribarri               | José Cenovio Urribarrí               | INE                  |
| VE231804 | Pedro Lucas Urribarri                | Pedro Lucas Urribarrí                | INE, marydn          |
| VE240109 | Naiguata                             | Naiguatá                             | marydn               |
| VE030402 | Santa Barbará                        | Santa Bárbara                        | tilde mal ubicada    |
| VE060701 | Santa Barbará                        | Santa Bárbara                        | tilde mal ubicada    |
| VE151203 | Santa Barbará                        | Santa Bárbara                        | tilde mal ubicada    |
| VE230503 | Santa Barbará                        | Santa Bárbara                        | tilde mal ubicada    |

### 4. Errores de tipeo y nombres truncados

Corregidos solo con evidencia externa (INE, prensa local, Wikipedia o el nombre
de la persona homenajeada).

| Código   | Antes                        | Después                       | Evidencia                                                             |
| -------- | ---------------------------- | ----------------------------- | --------------------------------------------------------------------- |
| VE1806   | Monseñor José Vicente de Und | Monseñor José Vicente de Unda | Truncado; la parroquia capital del municipio tiene el nombre completo |
| VE2107   | José Felipe Márquez Cañizal  | José Felipe Márquez Cañizalez | Truncado; marydn                                                      |
| VE1410   | Julio César Sala             | Julio César Salas             | Truncado; marydn                                                      |
| VE0508   | Mario Briceño Iragorri       | Mario Briceño Iragorry        | Apellido del escritor; la parroquia homónima ya dice Iragorry         |
| VE2308   | Jesús María Semprum          | Jesús María Semprún           | INE (censo 2011, Zulia)                                               |
| VE230801 | Jesús María Semprum          | Jesús María Semprún           | INE (censo 2011, Zulia)                                               |
| VE010115 | San Bernandino               | San Bernardino                | Error de tipeo; marydn                                                |
| VE112303 | Las Vegas Deltuy             | Las Vegas del Tuy             | Falta un espacio; marydn                                              |
| VE070204 | Guainiamo                    | Guaniamo                      | Río y pueblo Guaniamo; marydn                                         |
| VE071002 | Ariapo                       | Aripao                        | Pueblo Aripao; marydn                                                 |
| VE071005 | Motiaco                      | Moitaco                       | Pueblo Moitaco; marydn                                                |
| VE040402 | Arismendi                    | Aramendi                      | Prensa local y Wikipedia; prócer Francisco Aramendi                   |
| VE061003 | Palacios Fajardo             | Palacio Fajardo               | Prócer Manuel Palacio Fajardo; Wikipedia                              |
| VE210206 | General Rivas                | General Ribas                 | Prensa local (Diario de los Andes); José Félix Ribas                  |

## Problemas conocidos

- `VE171002` (Tubores): OCHA dice `Los Barales`, otras fuentes `Los Baleales`; sin fuente que desempate.
- Faltan parroquias en al menos 5 municipios (ej. Maroa, Julio César Salas, Francisco Javier Pulgar);
  los huecos en la numeración de códigos lo delatan.
- Algunos municipios de Amazonas incluyen el prefijo `Autónomo` en el nombre.
- La fuente refleja la división vigente al 2021-02-23; cambios posteriores deben verificarse.
- En municipios con varias parroquias, la parroquia capital lleva el nombre del municipio
  (convención del INE). Algunas fuentes la nombran por el pueblo capital (ej. Miranda/Timotes
  en Mérida, Diego Bautista Urbaneja/Lechería en Anzoátegui); sin verificar contra leyes estadales.
- No incluye capitales de municipios.

## Sitio web

Si cambia el número de registros, actualiza también `recordCounts` en
`web/src/config/datasets.ts`, que es lo que muestra el sitio.
