# Orby API

API do Orby (Go + MySQL): publicações, comentários, menções, salvos, conta privada, bloqueio, direct em tempo real (SSE) e notificações. O front fica em `orby-front`.

## Rodando

1. Crie o banco: `mysql -u root -p < sql/sql.sql` (**apaga e recria** as tabelas do banco `orby`).
2. Copie `.env.example` para `.env` e preencha.
3. `go run .`

## Testes

Os testes de banco usam um MySQL real e só rodam com `ORBY_TEST_DSN` apontando para um banco cujo nome termine em `_test` (as tabelas são recriadas a cada teste):

```sh
mysql -u root -p -e "CREATE DATABASE orby_test"
ORBY_TEST_DSN='usuario:senha@/orby_test?parseTime=True' go test ./...
```

Sem a variável, esses testes são pulados. O CI (`.github/workflows/ci.yml`) sobe um MySQL e roda tudo.
