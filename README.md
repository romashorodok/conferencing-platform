*Simplified version in other branch*

## Build

### Docker
Docker build utilizes BuildKit cache on the local machine, allowing it to send only pre-built containers, rather than resolving and building dependencies each time. This saves bandwidth, in my case I have around ~4GB of bandwidth/space for a media server.

**Deps:**
- [magefile](https://github.com/magefile/mage)
- Latest Docker version with [Buildx/BuildKit](https://github.com/docker/buildx) support

**Steps:** <br/> <br/>
**Copy docker-compose file** and **edit whatever you want**:
```bash
cp docker-compose.yml docker-compose-prod.yml && nvim docker-compose-prod.yml
```
**Build containers cache** with mage file:
```bash
mage buildx docker-compose-prod.yml
```
**Use context** (Skip this if not needed): <br/>
> Use your credentials 
```bash
docker context create prod --docker "host=ssh://root@<HOST_IP>,key=/Users/user/.ssh/id_rsa" && ssh-add /Users/user/.ssh/id_rsa
```
**Deploy images on context target**:
> Instead of `prod` may be used `default` context
```bash
mage buildxDeploy prod docker-compose-prod.yml
```
**Apply your docker compose**:
```bash
docker context use prod && \
  docker-compose -f docker-compose-prod.yml up -d && \
  docker context use default
```
### Dev
> [!IMPORTANT]
> To run need Docker Compose version 2.22 and later.

```bash
docker compose watch
```

Allow self-signed cert and open https://localhost:443

Login and password: ```root root```

### Migrations

**Deps:**
- [goose](https://github.com/pressly/goose)

Open [media-server](./media-server/)
```bash
make up
```

## Info
**Supported browsers**:
- Chrome 126 (later versions may behave differently)
- Firefox 126 (on Docker container, there is a 10-second picture loss)
- Android Chrome Mobile 126 (later versions may behave differently)
- Android Firefox 126 (only audio sending)
