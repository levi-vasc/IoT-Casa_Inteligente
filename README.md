# IoT---Rota-das-Coisas

Este projeto é a primeira etapa do problema de MI de Concorrência e Conectividade, semestre 2026.1, da Universidade Estadual de Feira de Santana (UEFS). Ele consiste na simulação de uma casa inteligente, na qual cada entidade (sensores, atuadores e demais componentes) atua de maneira independente, comunicando-se por meio de mecanismos de conectividade definidos pela aplicação. O objetivo é exercitar conceitos de concorrência, comunicação e coordenação entre componentes autônomos em um ambiente de IoT.

## Descrição do Projeto

O **IoT Casa Inteligente** é um projeto desenvolvido em **Go** (com suporte via **Docker**) para construir uma solução de automação residencial baseada em Internet das Coisas (IoT). O objetivo é integrar e controlar dispositivos e sensores de uma casa (iluminação, presença, temperatura, ar condicionado) por meio de uma aplicação central, permitindo monitoramento em **tempo real**, **automatizações** e **acionamentos remotos** de forma organizada e escalável.

Na simulação, cada entidade atua de maneira independente, com seu próprio ciclo de execução e responsabilidades, refletindo o comportamento típico de dispositivos IoT no mundo real. Essa independência permite exercitar, de forma prática, os principais desafios do domínio: concorrência (várias entidades executando simultaneamente), conectividade (troca de mensagens/dados entre componentes) e coordenação (garantir consistência e respostas corretas mesmo com eventos ocorrendo em paralelo).

## Arquitetura

A simulação segue um modelo de casa inteligente com gateway central: as **entidades** (sensores e atuadores) executam de forma independente e concorrente, enquanto o **servidor** atua como controlador e ponto de integração, recebendo mensagens via rede (**TCP/UDP**), mantendo o estado do ambiente e coordenando comandos/respostas.

### Componentes

  * Dispositivos
      * Sensores: produzem eventos e enviam ao gateway (presença e temperatura)
      * Atuadores: recebem comandos do gateway e alteram seu estado (lâmpadas e ares-condicionados)
      * Cada dispositivo roda em seu próprio fluxo de execução (goroutine)
  * Gateway/Servidor
      * Recebe e processa mensagens dos dispositivos via TCP/UDP
      * Mantém um estado consolidado (últimas leituras, estado de atuadores, disponibilidade/conexão).
      * Aplica regras e envia comandos aos atuadores
      * Funciona como a "ponte" entre conectividade e lógica do sistema   
