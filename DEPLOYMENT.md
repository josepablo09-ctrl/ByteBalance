# Despliegue en AWS

Guía para poner ByteBalance en producción. Los comandos usan la AWS CLI v2 y Docker. Los
valores entre `<…>` son tuyos (región, cuenta, dominios).

```
Navegador ──► CloudFront ──► S3 (frontend estático)
    │
    └──────► ALB/App Runner (HTTPS) ──► API Go (Docker) ──► RDS PostgreSQL
                                          │
                                          └─► Gemini / Telegram (salida a internet)
```

> **Una sola instancia de la API.** El bot de Telegram (long polling), el job de
> recordatorios y los límites de uso viven dentro del proceso. Dos instancias se
> pelearían por las actualizaciones del bot, enviarían recordatorios duplicados y
> duplicarían los límites. Despliega con **mínimo y máximo 1**.

## 0. Requisitos previos

- Una VPC con subredes públicas (para el ALB) y privadas (para la API y RDS), y un
  **NAT Gateway**: la API necesita salida a internet para Gemini y Telegram.
- Un certificado de **ACM** para el dominio de la API. Si usas un dominio propio para el
  frontend, el certificado de CloudFront debe estar en `us-east-1`.
- Un repositorio de **ECR** para cada imagen:

```bash
aws ecr create-repository --repository-name bytebalance-api
aws ecr create-repository --repository-name bytebalance-migrate
```

## a) PostgreSQL en RDS

1. Crea una instancia **RDS for PostgreSQL 16** con estas opciones:
   - **Subredes privadas**, sin acceso público.
   - **Almacenamiento cifrado** y **copias automáticas** (7 días o más).
   - **Multi-AZ** si necesitas alta disponibilidad.
2. Grupo de seguridad `rds-sg`: admite el puerto **5432 solo desde** el grupo de seguridad
   de la API (`api-sg`), nunca desde `0.0.0.0/0`.
3. En PostgreSQL 15 o superior, RDS ya exige TLS (`rds.force_ssl=1`). La API también lo
   exige fuera de desarrollo, así que la cadena de conexión lleva `sslmode=require`:

   ```
   postgres://<usuario>:<clave-url-encoded>@<endpoint-rds>:5432/bytebalance?sslmode=require
   ```

   Si la contraseña tiene caracteres como `@ : / ? #`, codifícalos en la URL (`%40`,
   `%3A`, etc.).

   Para verificar además el certificado del servidor, monta el bundle de CA de RDS en el
   contenedor y usa
   `sslmode=verify-full&sslrootcert=/certs/rds-global-bundle.pem`. El bundle se descarga
   de <https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem>.

4. **Migraciones.** Se aplican con la imagen `migrate` (stage del `Dockerfile`), antes de
   cada versión nueva de la API y desde una máquina con acceso a RDS (la instancia EC2 o
   una tarea de ECS en la VPC). Lee `DATABASE_URL` del entorno, así la contraseña no
   queda en la línea de comandos:

   ```bash
   docker run --rm --env-file /run/bytebalance.env <cuenta>.dkr.ecr.<región>.amazonaws.com/bytebalance-migrate:<versión>          # up
   docker run --rm --env-file /run/bytebalance.env <cuenta>.dkr.ecr.<región>.amazonaws.com/bytebalance-migrate:<versión> version
   ```

## b) Contenedor de la API

### Construir y publicar las imágenes

El `Dockerfile` es multi-stage:

- **Compilación:** `golang:1.27-alpine` genera un binario estático sin símbolos de
  depuración.
- **Imagen final:** `distroless/static:nonroot` (~25 MB), sin shell, como usuario sin
  privilegios y con `APP_ENV=production` por defecto.

```bash
REGISTRY=<cuenta>.dkr.ecr.<región>.amazonaws.com
VERSION=$(git rev-parse --short HEAD)
aws ecr get-login-password --region <región> | docker login --username AWS --password-stdin $REGISTRY

docker build --platform linux/amd64 -t $REGISTRY/bytebalance-api:$VERSION .
docker build --platform linux/amd64 --target migrate -t $REGISTRY/bytebalance-migrate:$VERSION .
docker push $REGISTRY/bytebalance-api:$VERSION
docker push $REGISTRY/bytebalance-migrate:$VERSION
```

### Opción recomendada: EC2 detrás de un ALB

1. Lanza una instancia **Amazon Linux 2023** `t3.small` (amd64, igual que el
   `--platform` del build). Para una `t4g` (ARM, más barata), compila con
   `--platform linux/arm64`:
   - en una **subred privada**, con el grupo de seguridad `api-sg` (puerto 8080 solo
     desde el grupo de seguridad del ALB);
   - con un **rol de IAM** de mínimo privilegio: lectura de ECR
     (`AmazonEC2ContainerRegistryReadOnly`), `secretsmanager:GetSecretValue` solo sobre
     el secreto de ByteBalance y `AmazonSSMManagedInstanceCore` para entrar por Session
     Manager sin abrir SSH.
2. Instala Docker: `sudo dnf install -y docker jq && sudo systemctl enable --now docker`.
3. Crea un **ALB** público con estas opciones:
   - Listener **HTTPS 443** con el certificado de ACM; redirige HTTP 80 a HTTPS.
   - Target group HTTP hacia la instancia en el puerto `8080`, con health check
     `GET /healthz` (devuelve 200 si la base responde).
   - **Idle timeout de 75 s**: el análisis con IA puede tardar hasta ~60 s.
4. Despliega una versión. El archivo de variables se genera en `/run`, que es memoria y
   no disco, con permisos solo para root:

   ```bash
   sudo sh -c 'aws secretsmanager get-secret-value --secret-id bytebalance/prod \
       --query SecretString --output text \
     | jq -r "to_entries[] | \"\(.key)=\(.value)\"" > /run/bytebalance.env && chmod 600 /run/bytebalance.env'

   docker run --rm --env-file /run/bytebalance.env $REGISTRY/bytebalance-migrate:$VERSION
   docker rm -f bytebalance-api 2>/dev/null   # detener la versión anterior: solo una instancia
   docker run -d --name bytebalance-api --restart unless-stopped \
     --env-file /run/bytebalance.env -p 8080:8080 \
     --log-driver awslogs --log-opt awslogs-region=<región> --log-opt awslogs-group=/bytebalance/api \
     $REGISTRY/bytebalance-api:$VERSION
   ```

   Los logs son JSON por stdout y nunca contienen secretos ni montos. Con `awslogs` van a
   CloudWatch: crea antes el grupo (`aws logs create-log-group --log-group-name
/bytebalance/api`) y da al rol permiso para escribir en él.

### Alternativa: App Runner

App Runner solo asigna CPU mientras atiende peticiones. En los periodos sin tráfico, el
bot de Telegram y los recordatorios dejarían de funcionar. Úsalo **solo si no usas
Telegram** (`TELEGRAM_BOT_TOKEN` vacía); si no, elige EC2.

- **Origen:** la imagen `bytebalance-api` de ECR, puerto `8080`, health check HTTP en
  `/healthz`.
- **Escalado:** mínimo y máximo **1 instancia**.
- **Red:** un **VPC connector** hacia las subredes privadas, para llegar a RDS. Todo el
  tráfico de salida pasa entonces por la VPC, así que necesita el **NAT Gateway** para
  llegar a Gemini.
- **Variables:** las secretas se referencian desde Secrets Manager; el rol de instancia
  necesita `secretsmanager:GetSecretValue` sobre ellas. Usa `TRUSTED_PROXY_HOPS=1`.

## c) Frontend en S3 + CloudFront

1. **Compila** con la URL pública de la API. El build falla si `VITE_API_URL` falta o no
   usa `https`, y genera una Content-Security-Policy que solo permite conectar con esa
   API:

   ```bash
   cd frontend
   npm ci
   VITE_API_URL=https://api.<tu-dominio>/api/v1 npm run build   # salida en frontend/build/
   ```

   Esa variable se incrusta en el bundle público: **nunca** pongas secretos en variables
   `VITE_*`.

2. Crea un **bucket S3 privado**, con _Block Public Access_ activado y sin hosting web
   estático.
3. Crea una **distribución de CloudFront** con estas opciones:
   - **Origen:** el bucket, con **Origin Access Control (OAC)**. Aplica la política de
     bucket que propone CloudFront, que solo le permite leer a la distribución.
   - **Default root object:** `index.html`; _Viewer protocol policy:_ Redirect HTTP to
     HTTPS.
   - **Páginas de error personalizadas:** respuestas **403 y 404 → `/index.html` con
     código 200**. Es el fallback de la SPA: rutas como `/habits` se resuelven en el
     navegador.
   - **Response headers policy** gestionada `SecurityHeadersPolicy`: HSTS, `nosniff`,
     `X-Frame-Options` y `Referrer-Policy`.
4. **Sube** el build. Los assets con hash se cachean un año; `index.html` y el resto se
   revalidan siempre:

   ```bash
   aws s3 sync build/_app/immutable s3://<bucket>/_app/immutable \
     --cache-control "public,max-age=31536000,immutable"
   aws s3 sync build s3://<bucket> --delete --exclude "_app/immutable/*" \
     --cache-control "no-cache"
   aws cloudfront create-invalidation --distribution-id <id> --paths "/index.html" "/_app/version.json"
   ```

5. Pon el dominio de CloudFront (o el tuyo) en `CORS_ALLOWED_ORIGINS` de la API (ver d).

## d) Variables de entorno en producción

Guarda todas las variables en **un secreto JSON** de Secrets Manager
(`bytebalance/prod`). No se ponen en la imagen, en el repositorio ni en el user data de
EC2. La API valida la configuración al arrancar y **no inicia** si falta una variable
obligatoria o si es insegura (base sin TLS, CORS sin https, llaves con formato inválido).

| Variable               | Producción                       | Notas                                                                                          |
| ---------------------- | -------------------------------- | ---------------------------------------------------------------------------------------------- |
| `APP_ENV`              | `production`                     | Ya es el valor por defecto de la imagen. Activa HSTS y exige TLS y https.                      |
| `APP_PORT`             | `8080`                           | Por defecto en la imagen.                                                                      |
| `DATABASE_URL`         | `postgres://…?sslmode=require`   | **Secreta.** Ver a).                                                                           |
| `JWT_SECRET`           | 48 bytes aleatorios              | **Secreta.** `openssl rand -base64 48`; distinta por entorno.                                  |
| `CORS_ALLOWED_ORIGINS` | `https://<dominio-del-frontend>` | Exactamente el origen de CloudFront o tu dominio, sin barra final.                             |
| `TRUSTED_PROXY_HOPS`   | `1`                              | Un ALB (o App Runner) delante. Con un valor mayor al real, un cliente podría falsificar su IP. |
| `GEMINI_API_KEY`       | llave de Google AI Studio        | **Secreta.** Opcional: vacía desactiva el análisis con IA.                                     |
| `GEMINI_MODEL`         | (vacía)                          | Opcional; por defecto `gemini-3.5-flash`.                                                      |
| `TELEGRAM_BOT_TOKEN`   | token de @BotFather              | **Secreta.** Opcional: vacía desactiva Telegram. Usa un bot distinto al de desarrollo.         |

`POSTGRES_*`, `GEMINI_API_URL` y `TELEGRAM_API_URL` son solo para desarrollo local; en
producción se dejan vacías.

Ejemplo del secreto, con valores ficticios:

```json
{
  "DATABASE_URL": "postgres://bytebalance:CLAVE@bytebalance.xxxx.<región>.rds.amazonaws.com:5432/bytebalance?sslmode=require",
  "JWT_SECRET": "<openssl rand -base64 48>",
  "CORS_ALLOWED_ORIGINS": "https://app.<tu-dominio>",
  "TRUSTED_PROXY_HOPS": "1",
  "GEMINI_API_KEY": "<llave>",
  "TELEGRAM_BOT_TOKEN": "<token>"
}
```

Si una llave o token se filtra, **rótalo** (Google AI Studio, @BotFather o un
`JWT_SECRET` nuevo) y actualiza el secreto. Cambiar `JWT_SECRET` cierra todas las
sesiones.

## Verificación tras el despliegue

```bash
curl -i https://api.<tu-dominio>/healthz   # 200 {"status":"ok"} y cabecera Strict-Transport-Security
```

En los logs de arranque deben aparecer `bytebalance api escuchando` con
`"env":"production"` y, si los configuraste, `telegram activado` y `análisis con IA
activado`. Abre el frontend, inicia sesión y revisa en la consola del navegador que no
haya errores de CORS ni de CSP.
