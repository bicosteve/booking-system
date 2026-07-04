# CI/CD Required GitHub Secrets

Set these under Settings -> Secrets and variables -> Actions.

| Secret | Purpose |
| --- | --- |
| DOCKERHUB_USERNAME | Docker Hub login / image namespace (value: bixoloo) |
| DOCKERHUB_TOKEN | Docker Hub access token |
| EC2_HOST | EC2 public host or IP |
| EC2_USER | SSH user (e.g. ubuntu) |
| EC2_SSH_KEY | Private SSH key for the EC2 user |
| EC2_ENV_FILE | Full prod env file contents for the app (DB_HOST, DB_USER, DB_PASSWORD, DB_PORT, DB_SCHEMA, REDIS_ADDRESS, REDIS_PORT, REDIS_DB, REDIS_PASSWORD, REDIS_NAME, RABBIT_HOST, RABBIT_PORT, RABBIT_USER, RABBIT_PASSWORD, RABBIT_VHOST, RABBIT_QUEUE, RABBITMQ_STATUS, KAFKA_STATUS, HTTP_PORT, ADMIN_PORT, CONTENT_TYPE, API_PATH, JWT_SECRET, SENDGRID_KEY, MAIL_FROM, AT_KEY, APP_USERNAME, PP_CLIENT_ID, PP_SECRET, STRIPE_NAME, STRIPE_SECRET, STRIPE_PUB_KEY, STRIPE_SUCCESS_URL, STRIPE_CANCEL_URL, LOGGER_FOLDER) |
| LOKI_URL | Loki push endpoint (e.g. https://<id>.grafana.net/loki/api/v1/push) |
| LOKI_USERNAME | Loki basic-auth user (Grafana Cloud instance/user id) |
| LOKI_PASSWORD | Loki basic-auth token / API key |

## Notes

- The EC2 server must have Docker and the docker compose plugin installed.
- The app image is bixoloo/booking-system, published on push to main as
  `latest` and the short git SHA.
- Manual deploy: Actions -> CI-CD -> Run workflow (must be run from the main
  branch), optionally set image_tag (defaults to latest).