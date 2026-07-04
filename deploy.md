### 1. Compile the code

```bash
 GOOS=linux GOARCH=amd64 go build -o ./bin/bookingapp ./cmd/main.go # mac & windows terminal
 go build -o ./bin/bookingapp ./cmd
```

### 2. SSH into the VM

```bash
	gcloud compute ssh vm-name --zone=vm-zone
```

### 3. Install Required packages

```bash
	sudo apt update && sudo apt upgrade -y
	sudo apt install nginx -y
```

### 4. Transfer files to the VM

```bash
   gcloud compute scp --recurse ./bin/bookingapp ./env-toml ./docs myvm:~ --zone=myzone
   # run this command where your project files are
```

### 5. Set up .toml

```bash

```

### 6. Creat the Log Folder

```bash
   sudo mkdir -p /var/log/booking-app
   sudo chown $USER:$USER /var/log/booking-app
```

### 7. Run the app

```bash
   ENV=prod ./bookingapp
```

### 8. Set up a 'systemd' service to keep the app running

```bash
   sudo nano /etc/systemd/system/booking-app.service

   [Unit]
   Description=Booking-App
   After=network.target

   [Service]
   ExecStart=/home/bico/bookingapp
   WorkingDirectory=/home/bico
   Restart=always
   Environment=ENV=prod
   User=bico

   [Install]
   WantedBy=multi-user.targe

   # enable it
   sudo systemctl deamon-reexec
   sudo systemctl enable bookingapp
   sudo systemctl start bookingapp

```

### 9. Configure Nginx as Reverse Proxy

```bash
   sudo nano /etc/nginx/sites-available/booking-app

   # settings
   server {
    listen 80;
    server_name 35.242.242.95;

    location / {
        proxy_pass http://localhost:7001;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /api/user/ {
        proxy_pass http://localhost:7001;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /api/admin/ {
        proxy_pass http://localhost:7002;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /docs/ {
        alias /home/bico/docs;
        index index.html;
    }
}

# Set active site configuration
# 1. remove default config
sudo rm /etc/nginx/sites-enabled/default

# 2. Link your site config
sudo ln -s /etc/nginx/sites-available/booking-app /etc/nginx/sites-enabled/

# 3. Test nginx configurations
sudo nginx -t

# 3b. Expected output
nginx: the configuration file /etc/nginx/nginx.conf syntax is ok
nginx: configuration file /etc/nginx/nginx.conf test is successful


# 4. Restart nginx to apply changes

sudo systemctl restart nginx

# 5. Check if nginx is listing to port 80
sudo netstat -tulnp | grep :80

# 5b. Expected
tcp        0      0 0.0.0.0:80      0.0.0.0:*       LISTEN      <pid>/nginx
```

---

## Production managed services

The app reads `ENV=prod` config from environment variables (see `controllers/base.go`).
For the production managed-service stack set these on the EC2 `.env`:

- **Kafka (Aiven) — SASL over TLS / SCRAM**
  - `KAFKA_STATUS=1`
  - `KAFKA_BROKER=<aiven-broker:port>`
  - `KAFKA_SASL_MECHANISM=SCRAM-SHA-256` (use `SCRAM-SHA-512` if your Aiven project requires it)
  - `KAFKA_SASL_USERNAME` / `KAFKA_SASL_PASSWORD`
  - `KAFKA_SECURITY_PROTOCOL=SASL_SSL`
  - `KAFKA_CA_PEM` — inline CA certificate PEM (preferred). Optionally `KAFKA_CA_LOCATION` to point at a mounted CA file instead.

- **Redis (Upstash) — TLS**
  - `REDIS_ADDRESS=<host>.upstash.io`, `REDIS_PORT=6379`, `REDIS_PASSWORD=<password>`
  - `REDIS_TLS=true` (default on in prod; can be set to `false` to force plaintext Redis)

- **RabbitMQ — off**
  - `RABBITMQ_STATUS=0` (no connection attempt)
  - To re-enable with AMQPS: set `RABBITMQ_STATUS=1`, `RABBIT_TLS=true`, and optionally `RABBIT_CA_PEM` / `RABBIT_CA_LOCATION` for private-CA brokers. Public-CA brokers work with no CA (system roots).

All boolean flags (`REDIS_TLS`, `RABBIT_TLS`) accept `true`/`false`/`1`/`0`.
The startup gate waits up to `STARTUP_DEPENDENCY_TIMEOUT` (default `60s`) for the enabled
dependencies to become reachable before starting the HTTP servers.
