### Datenbank Setup

```shell
podman run --name webappdb \
-p 54321:5432 \
-e POSTGRES_PASSWORD=n5wFfFtCm3 \
-e POSTGRES_USER=webapp \
-e POSTGRES_DB=webappdb \
-v webappdb:/var/lib/pgsql/data \
-d postgres:latest
```
