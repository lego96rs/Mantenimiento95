# Plan del Sistema

## Objetivo

Construir un sistema web de gestion, planificacion, ejecucion y publicacion de mantenimiento para operar desde navegador, siguiendo el patron tecnico de `pg-runner-master` en Go, pero adaptado al negocio de mantenimiento preventivo, correctivo, incidentes y trazabilidad operativa.

El sistema debe servir como fuente unica para:

- Definir activos, areas y componentes.
- Generar tareas de mantenimiento desde la documentacion tecnica existente.
- Programar actividades por frecuencia o por fecha.
- Asignar y desasignar tareas a tecnicos o responsables.
- Publicar incidentes y actividades operativas relacionadas con mantenimiento.
- Ejecutar, controlar, cerrar y auditar trabajos.
- Obtener historial y reportes por activo, area, estado y responsable.

## Vision funcional

La solucion no sera solo un calendario. Debe comportarse como un sistema de gestion de mantenimiento asistido por documentacion tecnica.

La idea central es:

1. Los manuales y planes de mantenimiento existentes en `docs/` son la fuente de tareas sugeridas.
2. Esas sugerencias se transforman en plantillas reutilizables dentro del sistema.
3. Las plantillas generan actividades programadas, tareas operativas, ordenes de trabajo o incidentes segun el caso.
4. Los responsables pueden asignar, reprogramar, desasignar, publicar y cerrar actividades.
5. Todo queda trazado con usuario, fecha, estado, comentarios y cambios.

## Fuente de tareas de mantenimiento

### Regla principal

Las tareas del sistema deben derivarse de las sugerencias de mantenimiento estipuladas en los archivos de `docs/`, no de una lista inventada manualmente.

### Fuentes documentales iniciales

Los siguientes documentos ya muestran tareas reales con frecuencia, objeto, problema y accion:

- `System Maintenance Plan.pdf`
- `Cleaning Instructions Streamline (Maintenance and Repair).pdf`
- `Open Shuttle 100b (Maintenance and Repair).pdf`
- `Roller Conveyor System STREAMLINE (Maintenance and Repair).pdf`
- `Belt Conveyor Streamline (Maintenance and Repair).pdf`
- `Belt Transfer Unit Streamline (Maintenance and Repair).pdf`
- `Local Control Cabinet 2.0 Streamline (Maintenance and Repair).pdf`
- `Pneumatic Control Panel for Warehouse Areas (Maintenance and Repair).pdf`
- `Pick-it-Easy Evo Work Station (Maintenance and Repair).pdf`
- `OSR Shuttle™ Evo 1D Elementary (Maintenance and Repair).pdf`

### Tipos de tareas detectadas en los manuales

Las tareas extraidas caen principalmente en estas categorias:

- Limpieza.
- Inspeccion visual.
- Verificacion funcional.
- Ajuste o reajuste.
- Reapriete.
- Lubricacion.
- Reemplazo por desgaste o falla.
- Control de posicion.
- Control de tension, centrado o alineacion.
- Reemplazo por personal calificado o por servicio especializado.

### Frecuencias detectadas

Las frecuencias de los manuales deben respetarse como base del sistema:

- `T`: diaria.
- `W`: semanal.
- `M`: mensual.
- `3M`: trimestral.
- `6M`: semestral.
- `12M`: anual.
- `B`: cuando aplique o por condicion.

### Regla de transformacion a tareas del sistema

Cada sugerencia documental debe convertirse en una plantilla de mantenimiento con:

- Activo o tipo de activo.
- Componente u objeto a revisar.
- Frecuencia base.
- Tipo de accion.
- Descripcion operativa clara.
- Criterio de ejecucion.
- Criterio de aceptacion o cierre.
- Accion esperada ante hallazgo.
- Nivel de criticidad.
- Si requiere personal calificado.

### Ejemplos representativos ya detectados

- Limpiar scanner en frecuencia semanal.
- Limpiar sensores opticos y reflectores en frecuencia mensual.
- Limpiar correas, rodillos y rieles en frecuencia semestral.
- Revisar boton de paro de emergencia en frecuencia anual.
- Ajustar alineacion de laser scanner cuando la orientacion sea incorrecta.
- Reapretar fijaciones de ruedas, motores o componentes cuando se detecte soltura.
- Reemplazar sensores, reflectores, correas, filtros o componentes defectuosos cuando la condicion lo indique.
- Drenar agua del separador de condensados en panel neumatico en frecuencia mensual.
- Ajustar la presion de operacion del panel neumatico a `6 bar`.

## Alcance detallado v1

La version inicial debe dejar operativa una plataforma de gestion de mantenimiento con enfoque realista y usable.

### 1. Autenticacion y seguridad

- Login local con usuario y contrasena.
- Sesiones server-side.
- CSRF en operaciones autenticadas.
- Cookies seguras, `HttpOnly` y `SameSite`.
- Rate limit de login.
- Security headers.
- Cambio forzado de contrasena para credenciales temporales.
- Middleware de autenticacion y autorizacion por rol.
- Registro de eventos de acceso y acciones sensibles.

### 2. Usuarios, roles y permisos

Roles base propuestos:

- `admin`: administra configuracion, usuarios, catalogos, planes, tareas, incidentes y reportes.
- `planner`: planifica actividades, publica tareas, asigna, desasigna, reprograma y revisa cumplimiento.
- `supervisor`: gestiona operacion diaria, redistribuye trabajo, valida incidentes y cierres.
- `technician`: ejecuta tareas asignadas, reporta avance, incidentes y cierre.
- `viewer`: lectura de tableros, historial y reportes sin edicion.

Permisos funcionales minimos:

- Crear, editar y desactivar usuarios.
- Crear y editar activos, ubicaciones y categorias.
- Crear y editar plantillas de mantenimiento.
- Programar actividades por fecha o frecuencia.
- Asignar y desasignar actividades.
- Cambiar responsable.
- Publicar incidentes.
- Publicar actividades para ejecucion.
- Registrar ejecucion, hallazgos y cierre.
- Reabrir tareas o incidentes si corresponde.

### 3. Maestro de activos y ubicaciones

El sistema debe manejar un maestro estructurado para soportar planificacion y trazabilidad:

- Activo.
- Codigo interno del activo.
- Tipo o familia.
- Area.
- Subarea o linea.
- Ubicacion fisica.
- Fabricante.
- Modelo.
- Numero de serie.
- Estado operativo.
- Criticidad.
- Documentos tecnicos asociados.

El activo debe permitir relacionarse con:

- Plantillas de mantenimiento.
- Tareas programadas.
- Ordenes de trabajo.
- Incidentes.
- Historial de ejecuciones.

### 4. Plantillas de mantenimiento

La plantilla sera la unidad maestra desde la cual se generan actividades.

Cada plantilla debe incluir:

- Nombre de la actividad.
- Activo o tipo de activo asociado.
- Documento fuente.
- Seccion o referencia documental.
- Frecuencia.
- Tipo de mantenimiento: preventivo, correctivo recurrente, inspeccion, limpieza, seguridad.
- Procedimiento resumido.
- Criterio de validacion.
- Si requiere checklist.
- Si requiere confirmacion de supervisor.
- Si requiere personal calificado.
- Prioridad.
- Tiempo estimado.

### 5. Programacion de tareas

El sistema debe permitir programar actividades desde plantillas o manualmente.

Casos minimos:

- Programacion diaria, semanal, mensual, trimestral, semestral y anual.
- Programacion por fecha puntual.
- Programacion por rango o ventana.
- Generacion automatica de tareas futuras segun frecuencia.
- Reprogramacion individual o masiva.
- Suspender temporalmente una plantilla sin eliminar historial.

Estados minimos de una tarea programada:

- Borrador.
- Programada.
- Publicada.
- Asignada.
- En progreso.
- Pausada.
- Completada.
- Cancelada.
- Vencida.
- Reprogramada.

### 6. Asignacion y desasignacion de tareas

El sistema debe contemplar la gestion operativa diaria de responsables.

Acciones requeridas:

- Asignar tarea a un tecnico especifico.
- Asignar tarea a un supervisor o area responsable.
- Desasignar tarea dejando trazabilidad del motivo.
- Reasignar tarea de un responsable a otro.
- Autoasignacion solo si el rol lo permite.
- Asignacion masiva por filtro.
- Bloqueo de asignacion si la tarea esta cerrada o cancelada.

Datos a registrar en cada movimiento:

- Usuario que asigna o desasigna.
- Responsable anterior.
- Responsable nuevo.
- Fecha y hora.
- Motivo del cambio.
- Comentario opcional.

### 7. Publicacion de actividades

El sistema no debe limitarse a crear tareas en backoffice; debe permitir publicarlas para ejecucion.

Publicar una actividad significa:

- Dejarla visible en la bandeja operativa de los responsables.
- Marcarla como lista para ejecutar.
- Registrar fecha de publicacion.
- Registrar usuario que publica.
- Permitir notacion de prioridad, ventana y notas operativas.

Casos de uso:

- Publicar mantenimiento preventivo de rutina.
- Publicar campaña de inspeccion.
- Publicar tarea extraordinaria por recomendacion tecnica.
- Publicar actividad correctiva derivada de incidente.

### 8. Incidentes de mantenimiento

El sistema debe manejar incidentes como entidad propia, no solo como comentario libre.

Un incidente debe permitir:

- Crear reporte manual de falla, anomalia o condicion insegura.
- Asociar activo, ubicacion y categoria.
- Clasificar severidad.
- Registrar descripcion, fecha y usuario reportante.
- Adjuntar observaciones de diagnostico.
- Convertir incidente en tarea u orden de trabajo.
- Asignar responsable.
- Escalar a supervisor.
- Cerrar o reabrir.

Estados minimos del incidente:

- Nuevo.
- Publicado.
- En revision.
- Asignado.
- En trabajo.
- Resuelto.
- Cerrado.
- Cancelado.

Tipos iniciales de incidente:

- Falla de equipo.
- Desalineacion o ajuste.
- Suciedad o contaminacion.
- Desgaste.
- Componente defectuoso.
- Riesgo de seguridad.
- Hallazgo durante inspeccion.

### 9. Ejecucion de actividades

La vista operativa debe permitir trabajar la actividad completa.

Cada ejecucion debe soportar:

- Inicio y fin.
- Estado actual.
- Responsable asignado.
- Checklist asociado.
- Observaciones.
- Hallazgos.
- Acciones realizadas.
- Repuestos o materiales usados.
- Tiempo invertido.
- Resultado final.
- Decision de cierre o escalamiento.

Resultados posibles:

- Ejecutada sin novedad.
- Ejecutada con observaciones.
- Ejecutada parcialmente.
- Requiere repuesto.
- Requiere servicio externo.
- No ejecutada por bloqueo operativo.
- Derivada a incidente o correctivo.

### 10. Publicacion y gestion de actividades extraordinarias

Ademas de la programacion por frecuencia, el sistema debe soportar actividades no recurrentes:

- Mantenimiento extraordinario.
- Campanas de inspeccion.
- Actividades por hallazgo.
- Actividades por auditoria.
- Actividades solicitadas por operacion.

Estas actividades pueden crearse:

- Desde una plantilla.
- Desde un incidente.
- Manualmente por usuario autorizado.

### 11. Checklists y criterios de cierre

Las actividades deben poder operar con checklist simple o estructurado.

Cada checklist puede incluir:

- Paso.
- Orden.
- Tipo de verificacion.
- Resultado esperado.
- Resultado registrado.
- Observacion.
- Obligatorio o no.

La tarea no debe cerrarse completamente si:

- Tiene pasos obligatorios pendientes.
- Tiene incidentes derivados sin resolver y la politica exige cierre supervisado.
- Falta validacion del supervisor en tareas criticas.

### 12. Historial, auditoria y trazabilidad

Todo cambio relevante debe quedar registrado.

Eventos a auditar:

- Creacion y edicion de plantillas.
- Programacion y reprogramacion.
- Publicacion de actividad.
- Asignacion, desasignacion y reasignacion.
- Inicio, pausa, cierre y cancelacion.
- Creacion y cierre de incidentes.
- Cambios de estado.
- Comentarios relevantes.

### 13. Reportes y seguimiento

La plataforma debe entregar visibilidad de gestion.

Reportes minimos:

- Cumplimiento por periodo.
- Cumplimiento por activo.
- Tareas pendientes.
- Tareas vencidas.
- Tareas por responsable.
- Incidentes por severidad y estado.
- Historial por activo.
- Historial por documento fuente.
- Actividades publicadas versus ejecutadas.

Exportaciones minimas:

- Tabla de tareas.
- Tabla de incidentes.
- Historial por activo.

## Reglas de negocio

### Reglas generales

- Toda accion sensible debe quedar ligada al usuario autenticado.
- Las fechas de programacion, publicacion, asignacion y cierre deben registrarse del lado servidor.
- Ningun rol obtiene permisos solo por interfaz; la autorizacion se valida en middleware y handlers.
- Toda plantilla de mantenimiento debe poder rastrearse a un documento fuente o a una decision administrativa justificada.
- La planificacion debe distinguir preventivo, correctivo, inspeccion, seguridad y extraordinario.

### Reglas para tareas basadas en manuales

- Una tarea recurrente no se crea libremente si ya existe una recomendacion equivalente en los manuales.
- El sistema debe permitir deduplicar tareas provenientes de distintas fuentes que describen la misma accion.
- La frecuencia documental se toma como frecuencia recomendada base, pero puede ajustarse por politica interna con trazabilidad.
- Las tareas `B` o "if required" no deben generarse automaticamente por calendario; deben tratarse como tareas condicionales o derivadas de inspeccion/incidente.

### Reglas para asignacion y desasignacion

- Solo `admin`, `planner` y `supervisor` pueden asignar o desasignar tareas.
- `technician` no puede desasignar tareas ajenas.
- Toda desasignacion requiere motivo.
- Toda reasignacion conserva historial completo del responsable anterior.

### Reglas para publicacion

- Una actividad programada puede mantenerse en borrador hasta validacion.
- Solo actividades publicadas aparecen como trabajo ejecutable para tecnicos.
- Publicar no equivale a cerrar ni asignar automaticamente, salvo politica expresa.

### Reglas para incidentes

- Un incidente puede existir sin tarea, pero debe poder derivarse a una.
- Un incidente critico debe quedar visible para supervisor o admin.
- El cierre de un incidente debe guardar causa, accion aplicada y resultado.

## Modelo funcional detallado

Entidades base propuestas:

- Usuario.
- Rol.
- Activo.
- Categoria de activo.
- Ubicacion.
- Documento tecnico.
- Plantilla de mantenimiento.
- Regla de frecuencia.
- Tarea programada.
- Publicacion de actividad.
- Asignacion.
- Orden de trabajo.
- Checklist.
- Item de checklist.
- Ejecucion.
- Observacion.
- Incidente.
- Historial de cambios.
- Bitacora de auditoria.

## Estructura objetivo del proyecto

```text
cmd/server/              entrada principal y wiring
internal/
  auth/                  password, sesiones, rate limit
  config/                configuracion por env y flags
  db/                    conexion SQLite, pragmas, migraciones
  middleware/            auth, roles, CSRF, logging, recovery
  models/                entidades y queries
  server/                handlers HTTP por modulo
  planner/               reglas de programacion, frecuencia y publicacion
  maintenance/           ejecucion, checklist, cierre y bitacora
  incidents/             incidentes, severidad, escalamiento y cierre
  assets/                activos, ubicaciones, categorias y documentos
  reports/               agregados y exportacion
web/
  templates/             layout y pantallas SSR
  static/
    css/
    js/
    img/
docs/
  plan/                  plan y decisiones
```

## Vistas funcionales esperadas

Navegacion principal sugerida:

- Inicio.
- Mis tareas.
- Calendario.
- Activos.
- Plantillas.
- Incidentes.
- Publicaciones.
- Reportes.
- Administracion.

Pantallas minimas:

- Login.
- Dashboard.
- Lista de tareas.
- Detalle de tarea.
- Lista de incidentes.
- Detalle de incidente.
- Calendario de actividades.
- Maestro de activos.
- Maestro de plantillas.
- Publicacion de actividades.
- Reportes.
- Administracion de usuarios y roles.

## Lineamientos de interfaz

- Seguir el patron visual de la referencia: layout comun, topbar, tarjetas, formularios limpios y lectura clara en celular y escritorio.
- Mantener login y shell privado con una experiencia coherente entre vistas operativas y administrativas.
- Evitar dependencias frontend complejas; priorizar SSR y JS progresivo.
- Diseñar navegacion por tareas reales: hoy, pendientes, calendario, activos, incidentes, publicaciones, reportes y admin.
- Destacar estados, prioridad, vencimiento y responsable.
- Dar acceso rapido a publicar, asignar, desasignar, reprogramar y cerrar.

## Criterios de exito de la fase plan

- Alcance funcional detallado y sin ambiguedades mayores.
- Arquitectura base alineada con la referencia.
- Tareas de mantenimiento documentadas como origen funcional del sistema.
- Reglas de programacion, asignacion, desasignacion, incidentes y publicacion definidas.
- Fases documentadas.
- Normas del repositorio y commits por fase definidas.
- Git inicializado.

## Fuera de alcance inmediato

- App movil nativa.
- Integraciones externas complejas.
- Motor avanzado de notificaciones multicanal.
- Multiempresa o multisitio complejo.
- IoT en tiempo real.
- Adjuntos pesados y gestion documental avanzada.
