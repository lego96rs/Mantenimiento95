# Plan Inicial del Sistema Web de Planificacion de Mantenimiento

Este proyecto parte desde cero y toma como referencia tecnica `C:\Users\LegionMFC\Documents\Proyecto\pg-runner-master`.

La meta es construir un sistema web de planificacion de mantenimiento con el mismo enfoque base del ejemplo:

- Backend en Go con `net/http` y estructura modular.
- Frontend SSR con `html/template` y JavaScript vanilla progresivo.
- Base de datos SQLite sin CGO.
- Login, sesiones, CSRF, headers de seguridad y rate limit.
- UI mobile-first y patron visual consistente en todas las vistas.

## Que se reutiliza de la referencia

- Estructura `cmd/`, `internal/`, `web/`, `docs/`.
- Seguridad del login y manejo de sesiones server-side.
- Middleware de autenticacion, autorizacion y CSRF.
- Sistema de templates con layout comun y assets embebidos.
- Roadmap por fases con entregables pequenos y demostrables.

## Que cambia para este proyecto

- El dominio ya no es registro de quiebres; ahora es planificacion y control de mantenimiento.
- El flujo principal pasa a ser plantillas, tareas programadas, publicaciones, incidentes, OT, checklist, activos y reportes.
- La administracion debe cubrir usuarios, roles, configuraciones y trazabilidad.
- Las tareas deben nacer desde sugerencias de mantenimiento reales presentes en los manuales y planes de `docs/`.
- La documentacion arranca primero para fijar alcance antes de codificar.

## Resultado esperado de la fase actual

- Quedar con una base documental en `docs/plan/`.
- Definir normas de trabajo en `agents.md`.
- Arrancar el repositorio con Git desde la fase `plan`.

## Documentos de esta carpeta

- `PLAN.md`: alcance funcional detallado, modulos, reglas de negocio y modelo operativo.
- `ROADMAP.md`: fases de trabajo y criterio de avance.
- `ARQUITECTURA.md`: mapeo entre la referencia y la nueva solucion.
