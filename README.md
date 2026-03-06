# Lab 5 — Server Side Rendering

Servidor HTTP hecho en Go sin usar net/http.
Implementa formularios HTML con POST y actualización con fetch() conectado a SQLite.

---

## Funcionalidades Obligatorias

- Manejo correcto de GET y POST
- Parseo manual de body (Content-Length)
- Inserción en SQLite
- Redirección 303 (POST/Redirect/GET)
- Botón +1 usando fetch()
- Tabla generada dinámicamente desde la base de datos
- Manejo de errores en DB
- Uso de defer rows.Close() y defer db.Close()

---

## Challenges Implementados

- Barra de progreso
- Marcar serie como COMPLETA
- Botón -1
- DELETE usando método DELETE
- Validación en servidor
- Sistema de rating (0–10) con tabla propia en SQLite

---

## Base de Datos

El archivo `series.db` contiene al menos 4 series precargadas.

---

## Screenshot

![Servidor corriendo](screenshot.png)
