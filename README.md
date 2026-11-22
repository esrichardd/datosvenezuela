# datosvenezuela

[![CI](https://github.com/esrichardd/datosvenezuela/actions/workflows/ci.yml/badge.svg)](https://github.com/esrichardd/datosvenezuela/actions/workflows/ci.yml)

API pública y gratuita con datos de Venezuela, pensada para desarrolladores.
Centraliza información que hoy está dispersa, desactualizada o difícil de consumir.

> 🚧 **Estado:** en desarrollo. La API ya sirve todos los datos de la v1: geografía,
> bancos, prefijos telefónicos y feriados (ver [`openapi.yaml`](openapi.yaml)). El sitio
> tiene lista la página de inicio, con una consola para probar la API; la documentación y
> la página de descargas vienen después.

## ¿Qué datos ofrece?

### v1 (en desarrollo)

- **Geografía:** estados (con sus capitales), municipios y parroquias, con códigos oficiales.
- **Bancos:** códigos de 4 dígitos, nombre comercial y nombre legal de los bancos activos.
- **Prefijos telefónicos:** códigos de área fijos con los estados que cubren, prefijos
  móviles por operadora y números no geográficos (0800, 0500…).
- **Feriados:** feriados nacionales según la ley (calculables para cualquier año) y el
  calendario bancario publicado por SUDEBAN.

Cada dataset documenta sus fuentes, criterios y correcciones en su propio `SOURCES.md`.

### Próximamente (ideas)

- Tasa de cambio oficial del BCV (USD y EUR) con historial diario
- Validación de RIF y cédula
- Constitución y leyes (texto consultable por artículo)
- Códigos postales

## Principios

- **Sin registro ni API key.** Cualquiera puede consumirla.
- **Solo lectura.** Todos los endpoints son `GET`.
- **Datos abiertos y trazables.** Cada dato indica su fuente y fecha de actualización.
- **Corregible por la comunidad.** Los datos de referencia viven en este repositorio;
  cualquier corrección se propone con un Pull Request.
- **Usable desde el navegador.** La API permite peticiones de cualquier origen (CORS), así
  que una página web puede consumirla directamente.
- **Uso justo.** Habrá un límite de peticiones por IP para mantener el servicio disponible
  para todos (aún no implementado).

## Stack

| Capa                  | Tecnología                                           |
| --------------------- | ---------------------------------------------------- |
| API                   | Go                                                   |
| Base de datos         | PostgreSQL (desde la v2; la v1 sirve CSV en memoria) |
| Sitio y documentación | Astro 7                                              |
| Infraestructura       | VPS + Docker Compose + Caddy, detrás de Cloudflare   |

## Estructura del repositorio

```
datosvenezuela/
├── api/            # Backend en Go
├── web/            # Sitio: documentación, playground y datasets (Astro)
├── data/           # Datos de referencia versionados (CSV), bajo CC BY 4.0
├── deploy/         # Despliegue en producción (Docker Compose, Caddy)
├── .github/        # Integración continua
├── openapi.yaml    # Contrato de la API (OpenAPI)
├── compose.yaml    # Entorno de desarrollo local
└── Makefile        # Comandos comunes del proyecto
```

> Las carpetas se irán creando a medida que el proyecto avance.

## Desarrollo local

### Requisitos

| Herramienta                                     | Para qué                       | Versión                                    |
| ----------------------------------------------- | ------------------------------ | ------------------------------------------ |
| [Go](https://go.dev/dl/)                        | API                            | la de `api/go.mod`                         |
| [golangci-lint](https://golangci-lint.run/)     | lint y formato del código Go   | v2                                         |
| [Node.js](https://nodejs.org/)                  | sitio                          | 24 (`web/.nvmrc`; con nvm: `nvm use`)      |
| pnpm, vía [Corepack](https://nodejs.org/api/corepack.html) | dependencias del sitio | la fijada en `web/package.json` (`packageManager`) |
| [lefthook](https://lefthook.dev/)               | git hooks                      | cualquiera reciente                        |

Corepack viene con Node.js y usa automáticamente la versión de pnpm del proyecto, sin
instalarlo global. Actívalo una vez, con la versión de Node del proyecto activa:

```bash
cd web && nvm use && corepack enable && cd ..
```

### Primeros pasos

```bash
make setup   # instala los git hooks y las dependencias del sitio
```

`make` sin argumentos lista todos los comandos, agrupados por componente: `api-*` para
la API y `web-*` para el sitio. Los comandos sin prefijo actúan sobre todo el proyecto:

```bash
make fmt     # formatea todo el código
make test    # corre todos los tests
make check   # todas las revisiones del proyecto (igual que CI); córrelo antes de subir cambios
```

### Levantar el proyecto

La API y el sitio corren por separado, cada uno en su terminal:

| Servicio | Comando        | URL                   |
| -------- | -------------- | --------------------- |
| API      | `make api-run` | http://localhost:8080 |
| Sitio    | `make web-dev` | http://localhost:4321 |

La consola "Prueba la API" del sitio llama a la API desde el navegador, así que para
usarla deben estar corriendo los dos.

### API (`api/`)

Lee los datos de `data/` al arrancar y los sirve desde memoria. Se configura con
variables de entorno:

| Variable   | Por defecto | Qué es                                     |
| ---------- | ----------- | ------------------------------------------ |
| `PORT`     | `8080`      | puerto HTTP                                |
| `DATA_DIR` | `../data`   | carpeta con los CSV (relativa a `api/`)    |

```bash
make api-test    # tests con el detector de condiciones de carrera
make api-check   # lint + tests + vulnerabilidades en dependencias
```

### Sitio (`web/`)

Se configura con variables de entorno. Todas tienen un valor por defecto que sirve para
desarrollo local; para cambiarlas, copia `web/.env.example` a `web/.env`:

| Variable         | Por defecto             | Qué es                                       |
| ---------------- | ----------------------- | -------------------------------------------- |
| `PUBLIC_API_URL` | `http://localhost:8080` | URL de la API a la que llama la consola      |

```bash
make web-build   # genera el sitio estático en web/dist
make web-test    # tests unitarios (Vitest)
make web-check   # formato + lint + tipos + tests
```

El mapa de Venezuela de la página de inicio se genera a partir de los límites oficiales
de OCHA; cómo regenerarlo está en [`web/src/data/README.md`](web/src/data/README.md).

## Contribuir

El proyecto está en desarrollo. Si quieres aportar ideas o fuentes de datos
confiables, abre un issue.

## Licencia

- **Código:** [MIT](LICENSE)
- **Datos:** [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/deed.es).
  Puedes usarlos libremente, incluso con fines comerciales, citando la fuente: _datosvenezuela_.
