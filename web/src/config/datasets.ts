/**
 * The datasets the API serves, as the site presents them (home and
 * /datasets). Written by hand: the data itself lives in ../data, and each
 * dataset documents its sources there in SOURCES.md.
 */

/**
 * Number of records per table.
 *
 * Keep in sync with the CSV files in ../data. When they change, update these
 * numbers (the SOURCES.md files remind you). They are kept apart so they can
 * later be generated from the CSV files without touching anything else.
 */
export const recordCounts = {
  states: 25,
  municipalities: 335,
  parishes: 1134,
  banks: 27,
  landlinePrefixes: 58,
  mobilePrefixes: 6,
  nonGeographicPrefixes: 4,
  nationalHolidays: 14,
  bankHolidays2026: 22,
} as const;

/** Color of the dataset's swatch. "pending" marks one not available yet. */
export type DatasetTone = "decor-1" | "decor-2" | "decor-3" | "fg" | "pending";

export interface Dataset {
  id: string;
  name: string;
  tone: DatasetTone;
  /** False for datasets announced but not served yet. */
  available: boolean;
  /** One line per table, e.g. "335 municipios". */
  records: { count?: number; label: string }[];
  /** Short summary of the source; the details are in SOURCES.md. */
  source: string;
  endpoints: string[];
}

export const datasets: Dataset[] = [
  {
    id: "geography",
    name: "Geografía",
    tone: "decor-1",
    available: true,
    records: [
      { count: recordCounts.states, label: "estados" },
      { count: recordCounts.municipalities, label: "municipios" },
      { count: recordCounts.parishes, label: "parroquias" },
    ],
    source:
      "Límites administrativos del INE, publicados por OCHA en HDX. Códigos P-code.",
    endpoints: [
      "/v1/states",
      "/v1/municipalities",
      "/v1/parishes",
      "/v1/places/search",
    ],
  },
  {
    id: "banks",
    name: "Bancos",
    tone: "decor-2",
    available: true,
    records: [{ count: recordCounts.banks, label: "códigos" }],
    source:
      "No hay lista oficial descargable: se cruzaron listas públicas y se verificó cada banco activo.",
    endpoints: ["/v1/banks"],
  },
  {
    id: "phones",
    name: "Prefijos",
    tone: "decor-3",
    available: true,
    records: [
      { count: recordCounts.landlinePrefixes, label: "fijos" },
      { count: recordCounts.mobilePrefixes, label: "móviles" },
      { count: recordCounts.nonGeographicPrefixes, label: "no geográficos" },
    ],
    source: "Plan nacional de numeración comunicado por CONATEL a la UIT.",
    endpoints: ["/v1/phone-prefixes", "/v1/states/{code}/phone-prefixes"],
  },
  {
    id: "holidays",
    name: "Feriados",
    tone: "fg",
    available: true,
    records: [
      {
        count: recordCounts.nationalHolidays,
        label: "nacionales, cualquier año",
      },
      { count: recordCounts.bankHolidays2026, label: "bancarios en 2026" },
    ],
    source:
      "LOTTT art. 184 y Ley de Fiestas Nacionales. Calendario bancario de SUDEBAN.",
    endpoints: ["/v1/holidays/national", "/v1/holidays/bank"],
  },
  {
    id: "exchange-rate",
    name: "Tasa BCV",
    tone: "pending",
    available: false,
    records: [{ label: "v2" }],
    source: "Banco Central de Venezuela, con historial diario.",
    endpoints: ["En desarrollo"],
  },
];
