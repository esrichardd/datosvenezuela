/** Site-wide settings: one place for names, URLs and navigation. */

const repoUrl = "https://github.com/esrichardd/datosvenezuela";

/** Internal routes, so a path is written in one place only. */
export const routes = {
  home: "/",
  docs: "/docs/",
  datasets: "/datasets/",
} as const;

export interface NavLink {
  label: string;
  href: string;
  external?: boolean;
}

export const site = {
  name: "datosvenezuela",
  description:
    "API pública y gratuita con datos de referencia de Venezuela: geografía, bancos, prefijos telefónicos y feriados.",
  lang: "es",
  repoUrl,
  author: {
    name: "Richard",
    // TODO: replace the LinkedIn and X placeholders with the real profiles.
    links: [
      { label: "GitHub", href: "https://github.com/esrichardd" },
      { label: "LinkedIn", href: "https://www.linkedin.com/in/TODO" },
      { label: "X", href: "https://x.com/TODO" },
    ],
  },
  nav: [
    { label: "Documentación", href: routes.docs },
    { label: "Datasets", href: routes.datasets },
    { label: "GitHub", href: repoUrl, external: true },
  ] satisfies NavLink[],
  license: "Código MIT · datos CC BY 4.0",
} as const;
