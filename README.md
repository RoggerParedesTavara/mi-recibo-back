# Mi Recibo

Sistema integral para la gestión y prorrateo de recibos de servicios básicos (luz, agua, gas) en inmuebles de alquiler con múltiples inquilinos.

---

## 📁 Estructura del Repositorio

```
mi-recibo/
├── back/     # Backend en Go + PostgreSQL 16 (API REST y servidor MCP)
├── front/    # Frontend de la aplicación (próximamente)
└── LICENSE   # Licencia MIT
```

---

## 🚀 Módulos

### 🔧 [Backend (`back/`)](./back/README.md)
Desarrollado en **Go** y **PostgreSQL 16**:
- **Gestión de entidades**: Propietarios, Casas e Inquilinos.
- **Lista dinámica de cargos extras**: Prorrateo equitativo ítem por ítem entre $N$ inquilinos activos.
- **Consumo estricto**: Cada inquilino paga únicamente su consumo medido.
- **Detección de sobreconsumo**: Alertas automáticas ante incrementos inusuales de consumo respecto al mes previo.
- **Servidor MCP**: Integración con agentes de Inteligencia Artificial (Claude Desktop, Antigravity, Cursor).
- **Documentación OpenAPI y Colección Postman**: Ver carpeta [`back/docs/`](./back/docs).

Consulta la documentación detallada del backend en **[`back/README.md`](./back/README.md)**.

---

### 💻 Frontend (`front/`)
*(En desarrollo)*
