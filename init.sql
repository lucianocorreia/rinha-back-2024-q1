CREATE TABLE IF NOT EXISTS "clientes" (
    "id" serial PRIMARY KEY NOT NULL,
    "nome" text NOT NULL,
    "saldo" integer DEFAULT 0 NOT NULL,
    "limite" integer DEFAULT 0 NOT NULL
);

CREATE INDEX clientes_id_idx ON "clientes" USING HASH(id);

CREATE TABLE IF NOT EXISTS "transacoes" (
    "id" serial PRIMARY KEY NOT NULL,
    "cliente_id" integer NOT NULL ,
    "valor" integer NOT NULL,
    "tipo" char(1) NOT NULL,
    "descricao" varchar(10) NOT NULL,
    "realizado_em" timestamp NOT NULL DEFAULT now()
);

CREATE INDEX transacoes_id_idx ON "transacoes" USING HASH(id);
CREATE INDEX transacoes_cliente_id_idx ON "transacoes" USING HASH(cliente_id);

create or replace procedure criar_tr(
    id_cliente INTEGER,
    valor integer,
    tipo text,
    descricao text,
    inout saldo_ret integer default null,
    inout limite_ret integer default null
)

    language plpgsql
as $$

begin
    UPDATE clientes
    set saldo = saldo + valor
    where id = id_cliente and saldo + valor >= - limite
    returning saldo, limite into saldo_ret, limite_ret;

    if saldo_ret is null or limite_ret is null then return; end if;

    commit;

    INSERT INTO transacoes (valor, tipo, descricao, cliente_id)
    VALUES (ABS(valor), tipo, descricao, id_cliente);
end;
$$;

DO $$
    BEGIN
        INSERT INTO clientes (nome, limite)
        VALUES
            ('cliente 01', 1000 * 100),
            ('cliente 02', 800 * 100),
            ('cliente 03', 10000 * 100),
            ('cliente 04', 100000 * 100),
            ('cliente 05', 5000 * 100);
    END; $$
