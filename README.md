# IoT — Casa Inteligente

Sistema IoT com gateway rodando nativamente em um PC Linux e sensores/atuadores/cliente em Docker.

## Início Rápido

1. **Iniciar o gateway no host:**
   ```bash
   ./scripts/run-gateway.sh
   ```

2. **Configurar e subir os containers:**
   ```bash
   cp .env.example .env   # ajuste GATEWAY_HOST se necessário
   docker compose up -d
   ```

Para instruções detalhadas de deployment, cenários de rede e configurações, consulte **[DEPLOY.md](DEPLOY.md)**.