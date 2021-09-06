# PedroBank

## Como lançar a aplicação

A aplicação foi feita e testada com Linux, e não foi testada com Windows.

A aplicação precisa de dois containers:

* ordepmfc/pedro-bank:pedro-bank-db
* ordepmfc/pedro-bank:pedro-bank-app

Ambos estão publicados no meu repositório do dockerhub:

[ordepmfc/pedro-bank](https://hub.docker.com/r/ordepmfc/pedro-bank/tags)

Para baixar as imagens, use:

```bash
sudo docker pull ordepmfc/pedro-bank:pedro-bank-db
sudo docker pull ordepmfc/pedro-bank:pedro-bank-app
```

Se preferir construir as imagens, use estes comandos, rodados a partir da raiz
deste projeto, onde se encontram os Dockerfiles.

```bash
sudo docker build -t local-pedro-bank-app -f Dockerfile-app .
sudo docker build -t local-pedro-bank-db -f Dockerfile-db .
```

Para simplificar, e por falta de tempo, não configurei os containers para
usarem uma rede própria, e usaremos a rede do host. Isto significa que as
as portas 8080 (do app) e 5432 (postgres) devem estar livres ao rodar os
containers.

**ATENÇÃO**
Certifique-se de que as portas 8080 e 5432 no host estejam livres!

### Lançar a DB

Para rodar o container postgres, use

```bash
sudo docker run -it --network host ordepmfc/pedro-bank:pedro-bank-db
```

A inicialização demora um pouco, pois os scripts de criação das tabelas serão
rodados automaticamente ao iniciar um container. Espere a DB se inicializar
antes de rodar o próximo container. Caso coloque o container como daemon,
com `-d` em vez de `-it`, espere um tempo até que a base de dados esteja
pronta.

Se quiser acessar a base de dados diretamente, e tiver psql instalado no
seu host use:

```bash
psql pedro_bank postgres -h localhost
```

### Lançar a aplicação

Para rodar a aplicação própria, use:

```bash
sudo docker run -it --network host ordepmfc/pedro-bank:pedro-bank-app
```

Caso tenha construído os containers a partir dos Dockerfiles, lembre-se de
ajustar o nome do container com o que foi dado no `-t` do `docker build`.

Se preferir rodar a aplicação direto do host, sem o container,
navegue até `src/`, e use:

```bash
go get pedro-bank/main
go run pedro-bank/main --certs ../certs
```

Note que aida é preciso que o container da db esteja rodando!

## Como usar a aplicação

Para usar aplicação, curl é uma opção. O servidor roda com TLS, usando um 
certificado auto-assinado de teste distribuído com o projeto.

A partir da raiz deste projeto, use:

```bash
curl -i https://localhost:8080/ping --request "GET" --cacert certs/cert.pem
```

Se não quiser se preocupar em verificar o certificado, use `-k`:

```bash
curl -i -k https://localhost:8080/ping --request "GET"
```

Esta rota, `/ping`, não está na especificação mas é útil para testar se o
servidor está rodando.

### Criar um usuário

```bash
curl -i -k https://localhost:8080/accounts --header "Content-Type: application/json" --request "POST" --data '{"name":"John Doe","CPF":"221.321-12","secret":"toto"}'
```

### Listar usuários

```bash
curl -i -k https://localhost:8080/accounts --request "GET"
```

### Obter o saldo da conta

```bash
curl -i -k https://localhost:8080/accounts/1/balance --request "GET"
```

Ajuste o id no URL caso o id do usuário não seja `1`.

### Login

```bash
curl -i -k https://localhost:8080/login --header "Content-Type: application/json" --request "POST" --data '{"CPF":"221.321-12", "secret":"toto"}'
```

Note o token na resposta:

```
HTTP/2 201 
content-type: text/plain; charset=utf-8
content-length: 29
date: Mon, 06 Sep 2021 03:55:23 GMT

{"token":"9e78d69a60e08c86"}
```

O valor será provavelmente diferente.

**ATENÇÃO**
O token expira em 2 minutos.

### Transferir

Começe por criar uma segunda conta.

```bash
curl -i -k https://localhost:8080/transfers --header "Authorization: 9e78d69a60e08c86" --header "Content-Type: application/json" --request "POST" --data '{"account_destination_id":2, "amount":34.72}'
```

Ajuste o token e o id de destino.

### Listar transferências

```bash
curl -i -k https://localhost:8080/transfers --header "Authorization: 9e78d69a60e08c86" --request "GET"
```

Ajuste o token.

## Como rodar os testes

Rode o container da aplicação executando o bash:

```bash
sudo docker run -it --network host ordepmfc/pedro-bank:pedro-bank-app /bin/bash
```

De dentro do container:

```bash
go test -v pedro-bank/server
```

O teste roda um servidor numa goroutine, e se conecta a ele mesmo.

**ATENÇÃO**
O teste apaga todas as contas e trasferências da DB.

Se preferir rodar os testes do host, sem o container, desligue o container do
app, navegue até o diretório `src/` deste projeto, e use:

```bash
go test -v pedro-bank/server
```

## Arquitetura e notas

O projeto usa o conector `pgx` para falar com a base de dados Postgres. Fora
isso, só usa a stdlib. Comecei usando `chi`, mas como as rotas não eram tão
complexas, optei por usar o pacote `http` da stdlib diretamente.

O módulo tem dois pacotes, `server`, com a lógica principal, e `main`, que
simplesmente inicia o servidor.

Arquivos principais no pacote `server`:

* server.go: Define as rotas inicia o servidor com a função Run()
* accounts.go: Define a lógica das rotas `/accounts`
* login.go: Define a lógica da rota `/login`
* transfers.go: Define a lógica da rota `/transfers`

Os usuários logados são mantidos em memória, num mapa, e não na base de dados.
O mapa mapea o token ao id do usuário logado, e o horário do login.

Os logins expiram a cada dois minutos. Uma goroutine definida em `login.go`
periodicamente limpa os logins expirados. A forma como está implementado não é
muito eficiente, pois a goroutine trava e atravessa todos os logins, então para
um servidor de produção seria necessário ter um índice ordenado pelo momento
de login, com o token da sessão, além do mapa de logins.

Ao usar receber um pedido de recurso protegido, o servidor verifica se o token
expirou e já remove a entrada do mapa de logins.

### Testes

Existe um pequeno teste unitário para a validação de criação de contas, mas o
teste principal, `real_test.go`, roda o servidor numa goroutine, e usa o
cliente http da stdlib para fazer requisições para o servidor que está rodando
no mesmo processo. Isso ajuda a fazer testes sem precisar fazer mocks de um
monte de coisas.