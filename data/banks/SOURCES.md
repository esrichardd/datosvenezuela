# Fuentes: bancos

## Qué contiene

| Archivo     | Registros | Columnas       |
| ----------- | --------- | -------------- |
| `banks.csv` | 27        | `code`, `name` |

- **`code`**: Código de Institución Bancaria de 4 dígitos que asigna el Banco Central
  de Venezuela para el Sistema de Cámara de Compensación Electrónica. Son los primeros
  4 dígitos de una cuenta bancaria de 20 dígitos. Es texto: el cero inicial es parte del código.
- **`name`**: nombre comercial de la institución.

**Criterio:** solo instituciones bancarias **activas**, más el Banco Central de Venezuela
(`0001`). No se incluyen nombre legal ni RIF: ninguna de las fuentes consultadas los trae.

## Fuentes

No existe una lista oficial descargable de códigos (ni del BCV ni de SUDEBAN).
La lista se construyó cruzando:

| Fuente                                                                                        | Fecha                 | Uso                                                               |
| --------------------------------------------------------------------------------------------- | --------------------- | ----------------------------------------------------------------- |
| [Gist de arodu](https://gist.github.com/arodu/af242b5e3d7fc4e4fb2710c76ec41fae)               | actualizado jun. 2025 | Lista base de códigos                                             |
| [Elyex](https://elyex.com/como-saber-cual-es-el-codigo-de-bancos-en-venezuela/)               | jun. 2022             | Segunda lista de códigos (incluye instituciones ya desaparecidas) |
| Informes mensuales del sistema bancario de [Banca y Negocios](https://www.bancaynegocios.com) | hasta abr. 2026       | Confirmar que cada banco sigue activo                             |
| Prensa (La Patilla, Wikipedia, iProfesional)                                                  | 2023–2026             | Fusiones, cambios de nombre y nuevas autorizaciones               |

Consultado el 2026-11-13.

## Cambios de nombre

| Código | Nombre anterior    | Nombre actual                     |
| ------ | ------------------ | --------------------------------- |
| 0169   | Mi Banco           | R4 Banco Microfinanciero          |
| 0175   | Banco Bicentenario | Banco Digital de los Trabajadores |

## Excluidos

| Código | Institución                            | Motivo                                                                |
| ------ | -------------------------------------- | --------------------------------------------------------------------- |
| 0116   | Banco Occidental de Descuento (BOD)    | Absorbido por el Banco Nacional de Crédito                            |
| 0149   | Banco del Pueblo Soberano              | No aparece en ninguna lista reciente                                  |
| 0190   | Citibank N.A.                          | Citigroup vendió su negocio en Venezuela al Banco Nacional de Crédito |
| 0601   | Instituto Municipal de Crédito Popular | Activo, pero no es un banco                                           |

## Problemas conocidos

- `0166` Banco Agrícola de Venezuela figura en los informes de 2026 con actividad mínima;
  se mantiene mientras siga reportado.
- Los códigos son estables, pero la lista de bancos cambia por fusiones y nuevas
  autorizaciones: debe revisarse periódicamente.
