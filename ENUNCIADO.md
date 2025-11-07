
## Contexto do problema

Nossa empresa precisa de um pequeno backend de “processamento de pedidos”, mas
com um fluxo mais realista do que apenas “recebe e grava”.

O fluxo é assim:
1. Um cliente faz uma requisição HTTP criando um pedido.
2. Esse pedido não é processado na hora: ele é enviado para uma fila.
3. Um serviço de processamento pega o pedido na fila e inicia o fluxo.
4. Parte desse fluxo depende de um enriquecimento assíncrono (por exemplo:
   buscar informações externas, calcular preços, validar cliente).
5. Quando o enriquecimento termina, o pedido é marcado como concluído.
6. Enquanto isso, o cliente pode consultar o status do pedido.

Você deverá dividir isso em serviços e rodar a infraestrutura com RabbitMQ,
PostgreSQL e containers (preferencialmente Kubernetes).

## Objetivo do desafio

Construir uma solução mínima composta por:

- Uma API em Go para criar e consultar pedidos.
- Um ou mais workers em Go que consomem mensagens do RabbitMQ e atualizam o
  banco.
- Uma infraestrutura containerizada contendo: API, RabbitMQ, PostgreSQL e
  (opcional) Redis.
- Manifestos Kubernetes (ou, como passo intermediário, docker-compose) para
  rodar tudo.

O foco é mostrar:

- domínio básico de Go para serviços;
- integração com RabbitMQ;
- acesso e modelagem simples em PostgreSQL;
- organização da infra.

## Requisitos funcionais

### Criação de pedidos (API)

- Endpoint: POST /orders
- Corpo esperado (exemplo):

    ```
    {
        "customer_id": "123",
        "items": [
            { "sku": "ABC", "qty": 2 },
            { "sku": "XYZ", "qty": 1 }
        ]
    }
    ```

- A API deve:

  - validar o payload;
  - gerar um `order_id` (UUID);
  - salvar o pedido no banco com status inicial “pending”;
  - publicar uma mensagem no RabbitMQ informando que o pedido foi criado;
  - retornar 201 ou 202 com o `order_id`.

### Idempotência

- A API deve aceitar o header X-Idempotency-Key.
- Se o mesmo header for enviado novamente, a API deve retornar o mesmo
  `order_id`, sem criar outro pedido.
- Para isso, crie uma tabela de controle de idempotência.

### Consulta de pedidos (API)

- Endpoint: GET /orders/{id}
- Deve retornar os dados do pedido e o status atual:

  - pending
  - processing
  - completed
  - failed

- Se o pedido tiver enriquecimento salvo, deve retornar também.

#### Processamento do pedido (Worker 1)

- Deve consumir a mensagem publicada pela API (ex.: routing key order.created).
- Deve atualizar o pedido para “processing”.
- Deve publicar uma nova mensagem solicitando o enriquecimento do pedido (ex.:
  order.enrichment.requested).

#### Enriquecimento (Worker 2)

- Deve consumir as mensagens de enriquecimento.
- Deve simular uma operação externa (ex.: aguardar alguns segundos ou chamar um
  serviço fake).
- Em caso de sucesso:

  - salvar os dados de enriquecimento numa tabela ligada ao pedido;
  - publicar uma mensagem order.enriched.

- Em caso de falha:

  - tentar novamente algumas vezes (retry);
  - ao estourar o limite de tentativas, mandar para uma fila de DLQ ou marcar o
    pedido como failed.

#### Finalização

- Um worker pode consumir order.enriched e atualizar o pedido para “completed”.

#### Autenticação simples

- A API deve exigir um token Bearer em pelo menos o endpoint de criação.
- O token pode vir de variável de ambiente.

## Requisitos não funcionais

- Código em Go organizado (módulos, pastas, separação de responsabilidades).
- Configuração via variáveis de ambiente (ex.: `DATABASE_URL`, `RABBITMQ_URL`,
  `SECRET_TOKEN`).
- Logs mínimos e claros.
- Uso de context e shutdown gracioso (capturar sinais) será considerado ponto extra.

## Infraestrutura esperada

Você deve disponibilizar manifestos Kubernetes (pasta k8s/) com:

- Deployments/StatefulSets para:

  - API
  - Worker de processamento
  - Worker de enriquecimento
  - PostgreSQL
  - RabbitMQ

- Services para expor a API e permitir comunicação interna entre os serviços.
- ConfigMaps/Secrets para as variáveis de ambiente.

Obs.: Se você preferir, pode entregar primeiro um docker-compose.yml com os
mesmos serviços. Porém, a versão em Kubernetes conta mais pontos, porque
queremos avaliar também sua familiaridade com orquestração.

## Modelagem sugerida do banco

Você pode adaptar, mas uma sugestão é:

```sql
create table orders (
    id uuid primary key,
    customer_id text not null,
    status text not null,
    total_amount numeric(12,2),
    created_at timestamp not null default now(),
    updated_at timestamp not null default now()
);

create table order_items (
    id serial primary key,
    order_id uuid not null references orders(id),
    sku text not null,
    qty int not null
);

create table order_enrichments (
    id serial primary key,
    order_id uuid not null references orders(id),
    data jsonb,
    created_at timestamp not null default now()
);

create table idempotency_keys (
    key text primary key,
    order_id uuid not null,
    created_at timestamp not null default now()
);
```

## O que entregar

1. Repositório (ou pasta) com o código Go da API e dos workers.
2. Manifestos Kubernetes (ou docker-compose) para subir:

   - API
   - RabbitMQ
   - PostgreSQL
   - Workers

3. Script SQL ou instruções de migração.
4. README.md com:

   - pré-requisitos (kubectl, kind/minikube ou docker);
   - como subir o ambiente;
   - exemplos de chamadas (curl) para criar e consultar pedidos;
   - como simular uma falha no enriquecimento;
   - como ver os registros no banco.

## Critérios de avaliação

- Funcionamento do fluxo fim a fim (criar → processar → enriquecer → concluir).
- Organização do código Go.
- Clareza da modelagem do banco.
- Uso correto do RabbitMQ (fila + exchange + routing key).
- Infraestrutura: se é possível subir o ambiente de forma reprodutível.
- Documentação: se outra pessoa consegue rodar seguindo o README.
- Pontos extras:
  - readiness/liveness probes no k8s
  - retries e DLQ
  - logs estruturados
  - cache para GET /orders/{id}

## Observações finais

- Você não precisa criar uma UI.
- Você não precisa usar framework web pesado em Go; pode ser net/http ou algo
  leve.
- Pode deixar o “serviço externo” de enriquecimento simulado.
- O importante é mostrar que você sabe conectar as peças, isolar
  responsabilidades e rodar isso em um ambiente de containers/orquestrador.

Boa sorte 👊
Fique à vontade para comentar no README qualquer decisão técnica que tomou ou
limitação de tempo. Isso também conta.
