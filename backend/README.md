# Backend — Buffalo Wings POS

API en Go (módulo `buffalo-pos`) para el POS y las pantallas de cocina (KDS).
Usa la librería estándar (`net/http`) y MySQL 8.0.

## Requisitos

| Herramienta | Versión | Para qué |
|---|---|---|
| Go | **1.21 o superior** (el proyecto compila con 1.26, ver abajo) | Compilar y correr la API |
| Docker + Docker Compose | cualquiera reciente | Levantar MySQL (`buffalo_mysql`) |

## Versión de Go y primera descarga del toolchain

`go.mod` declara `go 1.26.0`. **No se debe bajar esa versión:** algunas dependencias
exigen un Go reciente.

Desde Go 1.21, el comando `go` descarga solo el toolchain que pide `go.mod`
(mecanismo `GOTOOLCHAIN=auto`). Lo que esto significa para el equipo:

- **La primera vez** que alguien corra `go build`, `go run` o `go test`, Go descargará
  el toolchain 1.26 (unos 70 MB). Necesita internet y tarda un poco. Las siguientes
  veces ya está en caché (`~/go/pkg/mod/golang.org/toolchain@...`).
- **Go anterior a 1.21** no sabe descargar toolchains y falla con un error de versión.
  Hay que actualizar Go primero.
- **`toolchain not available`:** probablemente `GOTOOLCHAIN=local` o `GOFLAGS=-toolchain=local`
  están definidos. Se corrige con `export GOTOOLCHAIN=auto`.
- **Sin internet o detrás de un proxy:** la descarga falla. Instalar Go 1.26 a mano
  desde <https://go.dev/dl/>.
- **VS Code:** la extensión de Go puede mostrar errores mientras se descarga el toolchain.
  Al terminar, recargar la ventana (`Developer: Reload Window`).
- Para ver qué versión se está usando: `go version` dentro de `backend/`.

## Configuración

- Archivos YAML en `config/`: `app.yaml`, `security.yaml`, `business.yaml`.
- **Secretos por variables de entorno** (nunca en el código ni en el repo):
  - `JWT_SECRET`: obligatorio, mínimo 32 caracteres. Va en `backend/.env` (ignorado por git).
  - `DB_USER` y `DB_PASSWORD`: opcionales en local (hay valores por defecto de desarrollo).
- El archivo `.env.example` de la raíz muestra los nombres de las variables.

## Cómo correrlo

```bash
# 1. Base de datos (desde la raíz del repo)
docker compose up -d

# 2. API (desde backend/)
export JWT_SECRET="un-secreto-largo-de-al-menos-32-caracteres"
go run ./cmd/api
```

La API queda en `http://localhost:8080`. Prueba rápida:

```bash
curl http://localhost:8080/api/v1/healthz
```

## Comandos de calidad

Antes de cada commit, desde `backend/`:

```bash
gofmt -l -w .
go vet ./...
go build ./...
go test ./...
```

## Estructura (arquitectura hexagonal, un paquete por módulo)

```
internal/<modulo>/
  domain/             reglas del negocio puras (sin SQL ni HTTP)
  port/               interfaces (contratos)
  service/            casos de uso
  adapter/mysqlrepo/  implementación SQL
  adapter/httpapi/    handlers HTTP
internal/platform/    piezas compartidas (config, httpx, ratelimit)
cmd/api/main.go       arma todo y registra las rutas
migrations/           scripts SQL numerados; nunca se edita uno ya aplicado
```

La documentación de los endpoints está en [`../docs/API.md`](../docs/API.md).
