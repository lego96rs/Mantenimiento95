# Normas del Proyecto

## Objetivo de trabajo

Construir un sistema web de planificacion de mantenimiento tomando como referencia tecnica `C:\Users\LegionMFC\Documents\Proyecto\pg-runner-master`.

## Reglas base

- Priorizar funcionalidad real antes que sobreingenieria.
- Mantener el stack en Go y seguir el patron de la referencia.
- Reutilizar el enfoque de seguridad del proyecto base: login, sesiones, CSRF, cookies seguras y middleware por rol.
- Mantener interfaz SSR con `html/template`, CSS propio y JavaScript vanilla progresivo.
- Diseñar mobile-first, con un patron visual coherente entre login, modulo operativo y modulo admin.
- Toda logica sensible debe resolverse del lado servidor.
- Toda query debe ser parametrizada.
- Toda migracion se agrega como archivo nuevo.

## Normas de estructura

- `cmd/` para entrada de la aplicacion.
- `internal/` para logica de negocio, auth, db, middleware y modulos del dominio.
- `web/` para templates y assets.
- `docs/` para especificacion, plan y decisiones.
- Evitar handlers monoliticos y archivos gigantes.

## Normas de seguridad

- No confiar en datos del cliente para permisos, fechas, estado ni usuario.
- Autorizacion por middleware y validacion en handler.
- Password hashing fuerte y sesiones server-side.
- CSRF obligatorio en acciones autenticadas.
- Security headers activos desde el inicio.
- Rate limit en login y mensajes de error genericos.

## Normas de interfaz

- Seguir el patron visual del proyecto de referencia.
- Mantener el mismo criterio para login y areas privadas.
- Los textos visibles deben quedar en espanol.
- El codigo, identificadores y mensajes de commit deben ir en ingles.

## Normas de fase y commits

- Trabajar por fases cerradas y documentadas.
- Cada fase debe terminar con su propio commit.
- Cada fase debe commitearse solo despues de pasar sus tests o validaciones correspondientes.
- El commit de una fase se hace al cierre de la fase, una vez verificada funcionalmente.
- No mezclar una fase completa con otra en un mismo commit.
- Si una fase requiere varios commits, todos deben pertenecer claramente a la misma fase.
- La fase activa inicial es `plan`.

## Convencion de commits

Formato:

```text
phase(<fase>): <descripcion corta en ingles>
```

Ejemplos:

```text
phase(plan): bootstrap planning docs and project rules
phase(core): create server skeleton and project structure
phase(auth): add login, sessions and security middleware
phase(planning): add maintenance planning workflows
```

## Regla actual

El primer entregable del repositorio corresponde a la fase `plan`.
