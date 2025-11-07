# Teste técnico

## Como subir o ambiente

Você pode subir o ambiente localmente com Docker ou com KIND/Minikube.

### Deploy com Docker

Requisitos:
- docker

```sh
# Para subir os serviços localmente em background:
docker compose up -d

psql "host=localhost port=5432 user=admin password=asdf dbname=app" \
    -f ./migrations/001_initial_schema.sql
```

### Deploy com Kubernetes

Requisitos:
- Ambiente Kubernetes com uma CNI (ex.: Flannel, Cilium, Calico, etc) e com um
  StorageClass configurado.

<https://gitlab.c3sl.ufpr.br/yyvf22/k8s-test>

## Uso

### Exemplos de chamadas com cURL

#### Para criar um pedido

```sh
IDEMPOTENCY_KEY="$(uuid)"

curl -XPOST --location 'http://localhost:8080/orders' \
	-H 'Content-Type: application/json' \
	-H "X-Idempotency-Key: $IDEMPOTENCY_KEY" \
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

### Para consultar um pedido

Para consultar o pedido você precisa do ID do pedido. Esse ID é informado ao
criar o pedido.

```sh
# Para consultar um pedido com id $ID:
curl $HOST:$PORT/orders/$ID
```

### Como simular uma falha no enriquecimento do dado

Basta ter o caracter `f` no campo `customer_id`:

```sh
IDEMPOTENCY_KEY="$(uuid)"

curl -XPOST --location 'http://localhost:8080/orders' \
	-H 'Content-Type: application/json' \
	-H "X-Idempotency-Key: $IDEMPOTENCY_KEY" \
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
psql "host=localhost port=5432 user=admin password=asdf dbname=app"

# Para consultar as tabelas existentes:
app=# \dt

# Para consultar os registros da tabela X:
app=# SELECT * FROM X;
```

