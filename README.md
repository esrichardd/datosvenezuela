# apivenezuela

API pública y gratuita con datos de Venezuela, pensada para desarrolladores.
Centraliza información que hoy está dispersa, desactualizada o difícil de consumir.

> 🚧 **Estado:** en desarrollo. Los datos de la v1 ya están disponibles como CSV en
> [`data/`](data/); la API todavía no tiene endpoints.

## ¿Qué datos ofrece?

### v1 (en desarrollo)

- **Geografía:** estados (con sus capitales), municipios y parroquias, con códigos oficiales.
- **Bancos:** códigos de 4 dígitos, nombre comercial y nombre legal de los bancos activos.
- **Prefijos telefónicos:** códigos de área fijos con los estados que cubren, prefijos
  móviles por operadora y números no geográficos (0800, 0500…).

Cada dataset documenta sus fuentes, criterios y correcciones en su propio `SOURCES.md`.

### Próximamente (ideas)

- Tasa de cambio oficial del BCV (USD y EUR) con historial diario
- Validación de RIF y cédula
- Feriados nacionales
- Constitución y leyes (texto consultable por artículo)
- Códigos postales

## Principios

- **Sin registro ni API key.** Cualquiera puede consumirla.
- **Solo lectura.** Todos los endpoints son `GET`.
- **Datos abiertos y trazables.** Cada dato indica su fuente y fecha de actualización.
- **Corregible por la comunidad.** Los datos de referencia viven en este repositorio;
  cualquier corrección se propone con un Pull Request.
- **Uso justo.** Hay límite de peticiones por IP para mantener el servicio disponible para todos.

## Stack

| Capa                  | Tecnología                                         |
| --------------------- | -------------------------------------------------- |
| API                   | Go                                                 |
| Base de datos         | PostgreSQL                                         |
| Sitio y documentación | Astro                                              |
| Infraestructura       | VPS + Docker Compose + Caddy, detrás de Cloudflare |

## Estructura del repositorio

```
apivenezuela/
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

## Contribuir

El proyecto aún está en etapa de planificación. Si quieres aportar ideas o fuentes
de datos confiables, abre un issue.

## Licencia

- **Código:** [MIT](LICENSE)
- **Datos:** [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/deed.es).
  Puedes usarlos libremente, incluso con fines comerciales, citando la fuente: _apivenezuela_.
