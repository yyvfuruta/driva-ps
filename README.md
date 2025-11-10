# Teste técnico

## Como subir o ambiente

Você pode subir o ambiente localmente com Docker ou com KinD/Minikube.

### Deploy com Kubernetes

Requisitos:
- Docker
- KinD

```sh
# Para criar o cluster:
kind create cluster

# Aplicar os manifestos:
kubectl apply -k ./k8s

# Para acessar a API localhost:
kubectl port-forward svc/api 8080:8080
```

> [!NOTE]
> Com o deploy no Kubernetes, não precisa rodar as migrations. Ela vai rodar automaticamente.

### Deploy com Docker

Requisitos:
- Docker

```sh
# Para subir os serviços localmente em background:
docker compose up -d
```

### Execução das Migrations

```sh
docker exec -it my_postgres psql "host=localhost port=5432 user=admin password=asdf dbname=app"

# Crie as tabelas:
CREATE TABLE IF NOT EXISTS orders (
    id uuid PRIMARY KEY,
    customer_id TEXT NOT NULL,
    status TEXT NOT NULL,
    total_amount NUMERIC(12, 2),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS order_items (
    id SERIAL PRIMARY KEY,
    order_id uuid NOT NULL REFERENCES orders(id),
    sku TEXT NOT NULL,
    qty INT NOT NULL
);

CREATE TABLE IF NOT EXISTS order_enrichments (
    id SERIAL PRIMARY KEY,
    order_id uuid NOT NULL REFERENCES orders(id),
    data JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key TEXT PRIMARY KEY,
    order_id uuid NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

## Uso

### Exemplos de chamadas com cURL

#### Para criar um pedido

```sh
curl -XPOST --location 'http://localhost:8080/orders' \
	-H 'Content-Type: application/json' \
	-H "X-Idempotency-Key: asdf" \
	-H 'Authorization: Bearer cecd24ae-bbd6-11f0-bbc0-2c0da77e0ef5' \
	--data '{
		"customer_id": "123",
		"total_amount": 15.21,
		"items": [
			{
				"sku": "ABC",
				"qty": 2
			},
			{
				"sku": "XYZ",
				"qty": 1
			}
		]
	}'
```

#### Para consultar um pedido

Para consultar o pedido você precisa do ID do pedido. Esse ID é informado ao
criar o pedido.

```sh
# Para consultar um pedido com id $ID:
curl localhost:8080/orders/$ID
```

### Como simular uma falha no enriquecimento do dado

Basta ter o caracter `f` no campo `customer_id`:

```sh
curl -XPOST --location 'http://localhost:8080/orders' \
	-H 'Content-Type: application/json' \
	-H "X-Idempotency-Key: asdf" \
	-H 'Authorization: Bearer cecd24ae-bbd6-11f0-bbc0-2c0da77e0ef5' \
	--data '{
		"customer_id": "123f",
		"total_amount": 15.21,
		"items": [
			{
				"sku": "ABC",
				"qty": 2
			},
			{
				"sku": "XYZ",
				"qty": 1
			}
		]
	}'
```

## Como ver os registros do banco

```sh
# Para se conectar no banco:
kubectl exec -it po/postgres-0 -- psql "host=localhost port=5432 user=admin password=asdf dbname=app"

# Para consultar as tabelas existentes:
app=# \dt

# Para consultar os registros da tabela orders:
app=# SELECT * FROM orders;
```

## Como ver os registros no cache

Dentro do container do Redis:

```sh
kubectl exec -it pod/$NOME_DO_POD_DO_REDIS -- redis-cli

# Para listar todas as chaves:
> KEYS *

# Para ver o valor de uma chave:
> GET $KEY

# Para ver o TTL de uma chave:
> TTL $KEY
```
