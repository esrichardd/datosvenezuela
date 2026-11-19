# Fuentes: prefijos telefónicos

## Qué contiene

| Archivo                       | Registros | Columnas                           |
| ----------------------------- | --------- | ---------------------------------- |
| `landline_prefixes.csv`       | 58        | `code`                             |
| `landline_prefix_states.csv`  | 77        | `code`, `state_code`, `state_name` |
| `mobile_prefixes.csv`         | 6         | `code`, `operator`                 |
| `non_geographic_prefixes.csv` | 4         | `code`, `description`              |

Cada tipo de prefijo tiene su propio archivo porque cada uno describe cosas distintas:
los fijos se relacionan con estados, los móviles con una operadora y los no geográficos
con un servicio.

- **`code`**: código nacional de destino **sin el `0` inicial** (`212`, no `0212`). El `0`
  es el prefijo de marcación nacional, no parte del código; desde el exterior se marca
  `+58 212 …`. Es el formato del plan de numeración publicado ante la UIT (E.164).
  Un mismo código no se repite entre archivos.
- **`landline_prefix_states.csv`**: relación entre códigos fijos y estados (`state_code` de
  `../geography/states.csv`). Es **muchos a muchos**: un código puede cubrir varios
  estados (ej. `275`: Zulia, Mérida y Táchira) y un estado tener varios códigos.
- **`state_name`**: solo ayuda de lectura para quien edita el archivo a mano. La
  referencia real es `state_code`; `state_name` debe coincidir con el nombre en
  `../geography/states.csv`; la API lo valida al cargar los datos y no arranca si
  no coincide. Si un estado cambia de nombre, esta columna se regenera.
- **`operator`**: nombre comercial de la operadora móvil.

**Código de país:** `+58`. No se guarda en un CSV porque es un único valor.

## Fuentes

| Fuente                                                                                                                                           | Fecha      | Uso                                                      |
| ------------------------------------------------------------------------------------------------------------------------------------------------ | ---------- | -------------------------------------------------------- |
| [Plan nacional de numeración de Venezuela, comunicado por CONATEL a la UIT](https://www.itu.int/dms_pub/itu-t/oth/02/02/T02020000E30001PDFF.pdf) | 2011-04-08 | Códigos fijos con su cobertura, móviles y no geográficos |
| [Finanzas Digital: Digitel lanza nuevo prefijo 422](https://finanzasdigital.com/digitel-lanza-nuevo-prefijo-422-venezuela-lineas-moviles/)       | 2025       | Prefijo móvil `422` y prefijos actuales por operadora    |

Consultado el 2026-11-13.

## Transformaciones respecto a la fuente

1. La cobertura de la fuente dice "algunas ciudades de" uno o varios estados. Se registra
   solo la relación con el estado; no se indica qué ciudades.
2. `Estado Vargas` se registra como La Guaira (`VE24`), su nombre actual.
3. Se agrega `422` (Digitel, 2025), posterior a la comunicación de 2011.
4. Se usan nombres comerciales de operadoras: Movistar (TELCEL C.A.), Movilnet
   (Telecomunicaciones Movilnet C.A.), Digitel (Corporación Digitel C.A.).

## Excluidos

| Código | Motivo                                                                                          |
| ------ | ----------------------------------------------------------------------------------------------- |
| `415`  | Asignado a Globalstar de Venezuela (telefonía satelital); no se pudo confirmar que siga vigente |

## Problemas conocidos

- La lista de códigos fijos es de 2011. Los códigos geográficos cambian poco, pero
  debería contrastarse con una publicación más reciente de CONATEL.
- No se indica qué ciudades cubre cada código dentro de un estado.
