# uptime-reative

Pequeno serviço em Go que fica consultando o `/health` de outra aplicação e, **enquanto ela estiver saudável (resposta 2xx)**, chama um webhook a cada ciclo. Se ficar fora do ar (status não-2xx, timeout ou erro de conexão), para de chamar o webhook até voltar.

## Configuração (variáveis de ambiente)

| Variável         | Obrigatória | Padrão | Descrição                                   |
|------------------|-------------|--------|---------------------------------------------|
| `HEALTH_URL`     | sim         | –      | URL do health check (GET)                   |
| `WEBHOOK_URL`    | sim         | –      | URL chamada enquanto estiver saudável       |
| `WEBHOOK_METHOD` | não         | `POST` | Método HTTP do webhook                      |
| `CHECK_INTERVAL` | não         | `30s`  | Intervalo entre ciclos (ex.: `10s`, `1m`)   |
| `REQUEST_TIMEOUT`| não         | `10s`  | Timeout das requisições                     |
| `LOG_LEVEL`      | não         | `info` | `debug` mostra cada chamada                 |

## Rodando

```bash
docker run -d --restart unless-stopped \
  -e HEALTH_URL=https://minha-app.com/health \
  -e WEBHOOK_URL=https://meu-webhook.com/ping \
  -e CHECK_INTERVAL=30s \
  ghcr.io/carloslauriano/uptime-reative:latest
```

Localmente: `HEALTH_URL=... WEBHOOK_URL=... go run .`

## Build da imagem

O GitHub Actions (`.github/workflows/docker.yml`) roda os testes e publica a imagem multi-arch (amd64/arm64) em `ghcr.io/carloslauriano/uptime-reative` a cada push na `main` (`latest` e `sha-*`) e em tags `v*`.
