# Self-hosted Deploy

This directory drives the `deploy.yml` GitHub Actions workflow that runs on
the self-hosted runner `zgo-vps-cn` (8.219.77.159).

## Architecture

```
git push main → GH Actions self-hosted runner → docker build → docker compose up -d
```

- App container: `zgo-api` exposed on `:8025`
- Log viewer:    `dozzle` exposed on `:8081` (basic-auth protected)

## One-time server setup

The runner needs `/opt/zgo-deploy/dozzle/users.yml` to exist before the
first compose-up, otherwise Dozzle refuses to start.

```bash
ssh root@8.219.77.159
mkdir -p /opt/zgo-deploy/dozzle
docker run --rm amir20/dozzle:latest generate admin \
  --password '<your-password>' \
  --name 'Admin' \
  --email 'admin@zgi.ai' \
  > /opt/zgo-deploy/dozzle/users.yml
chmod 600 /opt/zgo-deploy/dozzle/users.yml
```

Then visit `http://8.219.77.159:8081` and log in.

## How developers see logs

- Open `http://8.219.77.159:8081`, log in with the credentials above
- All containers on the host show up, filter to `zgo-api` for the app
- Live tail, search, color-coded levels — no SSH needed

## Rolling back

The build step tags both `:sha-<short>` and `:latest`. To roll back:

```bash
docker tag zgo-api:sha-<older> zgo-api:latest
cd /opt/actions-runner/_work/zgo/zgo/deploy
docker compose up -d zgo-api
```
