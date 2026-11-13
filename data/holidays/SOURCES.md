# Fuentes: feriados

## Qué contiene

| Archivo               | Registros | Columnas                                  |
| --------------------- | --------- | ----------------------------------------- |
| `national_fixed.csv`  | 10        | `month`, `day`, `name`, `legal_basis`     |
| `national_easter.csv` | 4         | `days_from_easter`, `name`, `legal_basis` |
| `bank_2026.csv`       | 22        | `date`, `name`                            |

Venezuela tiene **dos calendarios** distintos:

- **Feriados nacionales (laborales)**: los fija la ley. Se guardan como **reglas**, no como
  fechas, para poder calcular cualquier año.
  - `national_fixed.csv`: feriados de fecha fija (ej. 5 de julio).
  - `national_easter.csv`: feriados que dependen de la Pascua. `days_from_easter` son los
    días de diferencia con el Domingo de Pascua (ej. Viernes Santo = `-2`). La fecha de
    Pascua se calcula con el algoritmo gregoriano estándar (Meeus/Jones/Butcher).
- **Feriados bancarios**: los publica SUDEBAN cada año (art. 68 de la Ley de Instituciones
  del Sector Bancario). Incluyen los nacionales más fiestas religiosas, que SUDEBAN suele
  trasladar al lunes, sin una regla fija (ej. en 2026 San José se mantiene en jueves). Por
  eso se guardan como **fechas concretas por año**: `bank_YYYY.csv` es el calendario completo
  publicado, incluidos los feriados nacionales que caen entre semana.

Los días de asueto que el Ejecutivo decreta sobre la marcha **no se incluyen**: no son
predecibles. Los domingos tampoco, por ser implícitos.

## Fuentes

| Fuente                                                                                                                                                             | Fecha      | Uso                                                                                   |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------- | ------------------------------------------------------------------------------------- |
| Ley Orgánica del Trabajo, los Trabajadores y las Trabajadoras (LOTTT), art. 184, vía [Acceso a la Justicia](https://accesoalajusticia.org/glossary/dias-festivos/) | 2012       | 1 de enero, Carnaval, Jueves y Viernes Santo, 1 de mayo, 24, 25 y 31 de diciembre     |
| [Ley de Fiestas Nacionales](https://venezuela.justia.com/federales/leyes/ley-de-fiestas-nacionales/gdoc), Gaceta Oficial N° 29.541                                 | 1971-06-22 | 19 de abril, 24 de junio, 5 de julio, 24 de julio y 12 de octubre                     |
| [Bloomberg Línea: lunes bancarios 2026](https://www.bloomberglinea.com/latinoamerica/venezuela/calendario-de-lunes-bancarios-en-venezuela-para-2026/)              | 2026       | Fechas efectivas (trasladadas a lunes) del calendario SUDEBAN 2026 y total de 22 días |
| [Finanzas Digital: calendario bancario 2026](https://finanzasdigital.com/sudeban-calendario-bancario-2026-feriados/)                                               | 2026       | Lista de feriados bancarios y base legal                                              |
| [Jibble: días feriados y bancarios 2026](https://www.jibble.io/es?p=388040)                                                                                        | 2026       | Contraste de fechas (incluye San José en jueves 19 de marzo)                          |

Consultado el 2026-11-13.

## Validaciones realizadas

- La Pascua de 2026 calculada (5 de abril) produce Carnaval el 16 y 17 de febrero y
  Semana Santa el 2 y 3 de abril, igual que el calendario publicado.
- Todos los feriados nacionales de 2026 que caen entre semana están en `bank_2026.csv`.
- `bank_2026.csv` tiene 22 fechas, todas entre semana, 11 de ellas en lunes: coincide con
  el total de 22 días y los 11 "lunes bancarios" reportados.

## Problemas conocidos

- Algunas fuentes listan los feriados religiosos en el día de la celebración y no en el día
  efectivo en que cierra la banca. Se usa la fecha efectiva.
- El calendario SUDEBAN de cada año debe agregarse cuando se publique (normalmente a fines
  del año anterior).
