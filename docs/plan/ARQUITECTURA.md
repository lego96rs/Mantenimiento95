# Arquitectura Base de Referencia

## Fuente

Referencia tecnica directa:

- `C:\Users\LegionMFC\Documents\Proyecto\pg-runner-master`

## Patron que se adopta

La solucion nueva debe reutilizar el mismo estilo de construccion:

- Go estandar con `net/http`.
- Server-side rendering con `html/template`.
- SQLite como almacenamiento local ligero.
- Seguridad integrada desde el inicio.
- Proyecto modular, simple de desplegar y sin dependencias de runtime complejas.

## Mapeo de la referencia al nuevo sistema

### Backend

Referencia:

- `cmd/server/main.go`
- `internal/server/`
- `internal/db/`
- `internal/middleware/`
- `internal/auth/`

Adaptacion:

- Mantener la misma separacion por capas.
- Crear paquetes nuevos para dominio de mantenimiento en vez de meter toda la logica en `server`.
- Centralizar reglas de programacion en un paquete dedicado, por ejemplo `internal/planner`.
- Separar incidentes, activos y ejecucion en paquetes de dominio propios para evitar mezclar programacion con operacion diaria.

### Seguridad

Referencia:

- Password hashing con Argon2id.
- Sesiones guardadas en base de datos.
- Cookie segura `HttpOnly` y `SameSite`.
- CSRF por sesion.
- Rate limit de login.
- Middleware `RequireUser` y `RequireAdmin`.

Adaptacion:

- Conservar el mismo esquema sin simplificarlo.
- Agregar matriz de roles para mantenimiento.
- Mantener la regla de que la UI no define permisos; el backend los impone.
- Aplicar permisos por accion sobre tareas, incidentes, asignaciones, publicaciones y cierres.

### Frontend

Referencia:

- `web/templates/layout.tmpl`
- `web/templates/login.tmpl`
- `web/static/css/app.css`

Adaptacion:

- Repetir el patron visual general: layout comun, topbar, paneles, formularios, tarjetas y navegacion por tabs o modulos.
- Disenar el login con la misma idea visual y el mismo flujo seguro.
- Mantener la interfaz mobile-first y despues ajustar escritorio.

### Datos

Referencia:

- Migraciones SQL numeradas.
- Estructuras de modelo claras y queries parametrizadas.

Adaptacion:

- Arrancar con migraciones desde el primer dia.
- Evitar cambios manuales fuera de migracion.
- Definir tablas base para usuarios, activos, documentos fuente, plantillas, tareas, publicaciones, asignaciones, incidentes, ejecuciones y auditoria.

## Decision de stack

- Backend: Go 1.23+.
- HTTP: `net/http`.
- DB: SQLite con `modernc.org/sqlite`.
- Templates: `html/template`.
- Assets: `embed.FS`.
- JS: vanilla, solo donde aporte valor.

## Criterios arquitectonicos

- Binario simple de desplegar.
- Seguridad por defecto.
- Codigo entendible y mantenible.
- Bajo costo operativo.
- Escalamiento funcional por modulos, no por complejidad de framework.
- Trazabilidad completa desde documento fuente hasta tarea ejecutada o incidente resuelto.
