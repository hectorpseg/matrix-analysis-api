# Factorización QR

Este proyecto recibe una matriz rectangular, calcula su factorización QR mediante reflexiones de Householder implementadas desde cero y devuelve las matrices Q, R y un conjunto de estadísticas calculadas sobre ambas.

El flujo principal es el siguiente:

1. El navegador o un cliente HTTP envía la matriz al API de Go.
2. Go calcula Q y R.
3. Go envía Q y R al API de Node.js para calcular estadísticas.
4. Node responde con las estadísticas.
5. Go devuelve Q, R y las estadísticas al cliente.
6. Una interfaz web simple demuestra todo el proceso.

## Arquitectura

```
Browser / Cliente HTTP
  -> Go / Fiber API (puerto 8080)
       -> Cálculo QR (Householder)
       -> HTTP POST al Node / Express API
            -> Estadísticas de Q y R
       <- Respuesta
  <- Respuesta con Q, R y stats
```

- **Go + Fiber**: expone `POST /api/v1/qr` y sirve el frontend.
- **Node.js + Express**: expone `POST /api/v1/stats`.
- **Comunicación HTTP**: el servicio Go llama al servicio Node a través de la variable `NODE_API_URL`.
- **Docker**: ambos servicios están dockerizados y se orquestan con `docker compose`.
- **Frontend**: es un único archivo HTML embebido en el binario de Go y servido en `/`.

## Factorización QR

- Q es una matriz ortogonal (`Q^T * Q ≈ I`).
- R es una matriz triangular superior.
- Se cumple `A ≈ Q * R`.
- La implementación usa reflexiones de Householder.
- Soporta matrices cuadradas, altas (`m > n`) y anchas (`m < n`).

La factorización QR tiene ambigüedad de signo: los valores de Q y R pueden diferir en signo respecto a otra implementación válida y aun así satisfacer `A = Q * R`, `Q^T * Q = I` y `R` triangular superior. Esto es correcto y esperado.

Nota sobre el enunciado del reto: la descripción original menciona "rotar" la matriz en la sección de arquitectura, pero los requerimientos funcionales piden explícitamente una factorización QR. Esta implementación sigue el requerimiento funcional de factorización QR.

## API

### Go API

#### `POST /api/v1/qr`

Calcula la factorización QR de la matriz recibida y consulta las estadísticas al servicio Node.

Request:

```json
{
  "matrix": [
    [1, 2],
    [3, 4]
  ]
}
```

Response (`200 OK`) para la matriz de ejemplo:

```json
{
  "q": [
    [-0.3162277660168382, 0.948683298050514],
    [-0.9486832980505137, -0.3162277660168381]
  ],
  "r": [
    [-3.16227766016838, -4.427188724235732],
    [0, 0.6324555320336751]
  ],
  "stats": {
    "global": {
      "max": 0.948683298050514,
      "min": -4.427188724235732,
      "average": -0.9486832980505141,
      "sum": -7.589466384404113
    },
    "q": { "isDiagonal": false },
    "r": { "isDiagonal": false }
  }
}
```

Los valores numéricos pueden variar en signo debido a la ambigüedad de signo propia de la factorización QR.

#### `GET /health`

Devuelve el estado del servicio:

```json
{ "status": "ok" }
```

### Node API

#### `POST /api/v1/stats`

Recibe Q y R y devuelve estadísticas globales y diagnósticos de diagonalidad.

Request:

```json
{
  "q": [[...], [...]],
  "r": [[...], [...]]
}
```

Response (`200 OK`):

```json
{
  "global": {
    "max": 8,
    "min": 1,
    "average": 4.5,
    "sum": 36
  },
  "q": { "isDiagonal": false },
  "r": { "isDiagonal": true }
}
```

Cálculos incluidos:

- Máximo global de todos los valores de Q y R.
- Mínimo global.
- Promedio global.
- Suma global.
- Si Q es diagonal.
- Si R es diagonal.

#### `GET /health`

```json
{ "status": "ok" }
```

### Validación

El API de Go valida la matriz de entrada antes de calcular QR. Rechaza con `400 Bad Request` (`{"error":"invalid matrix"}`):

- Matriz vacía o con filas vacías.
- Filas de longitudes desiguales (matriz irregular o "ragged").
- Valores no finitos (`NaN` o `Inf`).
- JSON mal formado o campo `matrix` incorrecto.

El API de Node valida Q y R de forma similar y devuelve `400 Bad Request` (`{"error": "..."}`).

La diagonalidad se evalúa con una tolerancia de `1e-9`: un valor fuera de la diagonal cuyo valor absoluto sea menor o igual a `1e-9` se considera cero.

## Ejecución local

Levantar todo el proyecto con Docker Compose:

```bash
docker compose up --build
```

- Go API: http://localhost:8080
- Health Go API: http://localhost:8080/health
- Node API: se comunica internamente en `http://node-api:3001`; no está expuesto por defecto.

Para ejecutar las pruebas directamente:

```bash
# Go
cd go-api
go test ./...

# Node.js
cd node-api
npm test
```

## Tests

El repositorio incluye pruebas automatizadas en ambos servicios.

**Go (`go-api`)**

- `main_test.go`: health endpoint, handler QR con matriz válida, matrices inválidas, respuestas de error del servicio Node, servicio Node no disponible y timeout.
- `qr_test.go`: propiedades matemáticas de la factorización QR (`Q * R ≈ A`, `Q^T * Q ≈ I`, `R` triangular superior) para matrices cuadradas, altas, anchas, rank-deficientes, diagonales, de una sola fila y con columnas nulas; también valida entradas inválidas.

**Node.js (`node-api`)**

- `test/health.test.js`: endpoint `/health`.
- `test/stats.test.js`: validación de matrices, estadísticas globales sobre matrices cuadradas y rectangulares, detección de diagonalidad con tolerancia `1e-9`, y la estructura completa de `computeStats`.

## Frontend

El frontend es un archivo HTML simple embebido en `go-api/web/index.html` y servido en la raíz (`/`). Permite:

- Ingresar una matriz (filas separadas por saltos de línea, valores por espacios).
- Calcular su factorización QR.
- Visualizar A, Q y R.
- Visualizar el producto `Q * R`.
- Ver las estadísticas calculadas.

No usa frameworks ni dependencias de frontend; es HTML, CSS y JavaScript vanilla.

## Deployment

La configuración actual está pensada para Docker Compose con dos servicios:

- `go-api`
- `node-api`

El servicio Go se conecta al servicio Node usando la variable de entorno `NODE_API_URL`. En `docker-compose.yml` este valor es `http://node-api:3001`.

URL pública del servicio Go: [agregar URL]
URL pública del servicio Node: [agregar URL]

## Decisiones técnicas

- **Householder desde cero**: no se usó una librería de álgebra lineal para la factorización QR.
- **QR completa**: Q es `m x m` y R es `m x n`, lo que permite soportar matrices rectangulares.
- **Node solo calcula estadísticas**: mantiene separada la responsabilidad y mantiene el servicio Go como orquestador.
- **HTTP entre servicios**: simple y alineado con la arquitectura del reto.
- **Dockerización**: facilita la ejecución local y el despliegue.
- **Dependencias mínimas**: Express en Node y Fiber en Go; el frontend no tiene dependencias.

## Limitaciones / supuestos

- El frontend y el API de Go esperan matrices de valores numéricos finitos.
- La comunicación entre servicios usa HTTP sin retry; si Node falla o tarda más de 10 segundos, Go devuelve `502 Bad Gateway`.
- La diagonalidad se evalúa con tolerancia `1e-9` y solo tiene sentido para matrices cuadradas; matrices no cuadradas se reportan como no diagonales.
- La factorización QR tiene ambigüedad de signo, por lo que los resultados numéricos pueden variar en signo respecto a otras implementaciones.

## Ejemplo rápido

```bash
curl -X POST <GO_API_URL>/api/v1/qr \
  -H "Content-Type: application/json" \
  -d '{"matrix":[[1,2],[3,4]]}'
```

Reemplaza `<GO_API_URL>` por la URL local (`http://localhost:8080`) o por la URL pública correspondiente.
