# Roadmap por Fases

Cada fase debe cerrar con algo revisable, versionable y con commit propio.

## Fase 0 - Plan

- Crear estructura documental inicial.
- Revisar `pg-runner-master` y fijar la arquitectura de referencia.
- Definir alcance v1, modulos, roles y restricciones.
- Documentar que las tareas deben originarse desde manuales y planes de mantenimiento existentes en `docs/`.
- Definir reglas de asignacion, desasignacion, publicacion, programacion e incidentes.
- Crear `agents.md` con normas de ejecucion.
- Inicializar Git.

Commit sugerido:

```text
phase(plan): bootstrap planning docs and project rules
```

## Fase 1 - Esqueleto base

- Crear modulo Go.
- Montar `cmd/server`, `internal`, `web`.
- Configuracion por variables de entorno.
- Servidor HTTP base con `healthz`.
- Templates y static embebidos.
- Conexion SQLite y migraciones.

Commit sugerido:

```text
phase(core): create server skeleton and project structure
```

## Fase 2 - Seguridad y acceso

- Hash de contrasenas.
- Login y logout.
- Sesiones server-side.
- CSRF y security headers.
- Middleware por rol.
- Bootstrap del primer admin.

Commit sugerido:

```text
phase(auth): add login, sessions and security middleware
```

## Fase 3 - Activos y catalogos

- CRUD de activos.
- Areas, categorias y criticidad.
- Relacion entre activos y documentos.
- Registro de documentos fuente y su trazabilidad con plantillas de mantenimiento.
- Busqueda y filtros.

Commit sugerido:

```text
phase(assets): add asset master data and classification
```

## Fase 4 - Planificacion

- CRUD de planes y plantillas de mantenimiento.
- Carga de plantillas derivadas de manuales.
- Frecuencias y reglas de programacion.
- Generacion de tareas, publicaciones y OT.
- Vista calendario o agenda.
- Programacion manual y recurrente.

Commit sugerido:

```text
phase(planning): add maintenance planning workflows
```

## Fase 5 - Ejecucion operativa

- Bandeja de tareas pendientes.
- Asignacion, desasignacion y reasignacion.
- Publicacion de actividades para ejecucion.
- Checklists por tipo de mantenimiento.
- Registro de ejecucion, observaciones y tiempos.
- Incidentes, escalamiento y cierre.
- Cierre y estados.

Commit sugerido:

```text
phase(execution): add work orders and execution tracking
```

## Fase 6 - Reportes

- Cumplimiento por periodo.
- Historial por activo.
- Incidentes por severidad y estado.
- Actividades publicadas versus ejecutadas.
- Pendientes y atrasos.
- Exportacion de reportes.

Commit sugerido:

```text
phase(reports): add dashboards and exportable reports
```

## Fase 7 - Hardening y deploy

- Pruebas funcionales y de seguridad.
- Hardening de cabeceras, sesiones y validaciones.
- Build para Linux.
- Runbook de despliegue y respaldo.

Commit sugerido:

```text
phase(release): harden application and document deployment
```

## Regla de avance

- No mezclar dos fases grandes en un mismo commit.
- Si una fase crece mucho, dividir en subcommits coherentes sin romper la regla principal.
- No iniciar la siguiente fase sin dejar la anterior documentada y funcional.
