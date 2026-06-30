# Como rodar os testes de aceitação (acc tests)

Guia prático e direto para executar os testes `TestAcc*` deste provider. Para a
filosofia de *escrita* dos testes (padrões, cenários, sweepers) veja
[UNIT_TESTING.md](UNIT_TESTING.md) e os memos em `.claude/`.

## TL;DR

```bash
# 1. Replay — não precisa de infra, nuvem nem segredos. Roda das cassettes commitadas.
make testacc-replay

# 2. Record — grava cassettes novas contra uma API de verdade (fake ou produção).
MGC_ENDPOINT=https://... MGC_API_KEY=<uuid> make testacc-record

# 3. Live — bate na API real sem gravar nada.
MGC_ENDPOINT=https://... MGC_API_KEY=<uuid> MGC_VCR_MODE=off make testacc
```

> O dia a dia é `make testacc-replay`: é o que roda em CI, não custa recurso e
> não precisa de credencial.

## Como funciona (em 30 segundos)

Os acc tests são **endpoint-agnósticos**: o teste não sabe se está falando com
uma API fake ou com a produção — quem decide é a env `MGC_ENDPOINT`, injetada no
bloco `endpoints` do provider pelo harness em `mgc/internal/acctest`.

Por cima disso há uma camada de **VCR** ([go-vcr](https://gopkg.in/dnaeon/go-vcr.v4)):
cada teste grava/reproduz suas chamadas HTTP em uma *cassette*
(`mgc/<service>/testdata/cassettes/<NomeDoTeste>.yaml`). É isso que permite rodar
a suíte inteira sem nuvem. O modo do VCR é escolhido pela env `MGC_VCR_MODE`.

## Pré-requisitos

| Item | Obrigatório quando | Observação |
|------|--------------------|------------|
| Go (versão do `go.mod`) | sempre | |
| Binário do `terraform` no `PATH` | sempre | sem ele os testes são **skipados** (não falham). Alternativa: `TF_ACC_TERRAFORM_PATH` ou `TF_ACC_TERRAFORM_VERSION`. |
| `MGC_ENDPOINT` + `MGC_API_KEY` | record e live | no replay têm defaults |
| Uma API alvo (fake ou prod) | record e live | replay não toca rede |

`TF_ACC=1` é o que liga os testes de aceitação — sem ela qualquer `go test`
**pula** os `TestAcc*`. Os alvos do `make` já setam isso pra você.

## Os três modos de VCR

A env `MGC_VCR_MODE` controla o recorder (ver `mgc/internal/acctest/vcr.go`):

| `MGC_VCR_MODE` | Comportamento | Precisa de rede/API? |
|----------------|---------------|----------------------|
| `replay` | só reproduz; **falha** se faltar cassette. É o modo de CI. | Não |
| `record` | sempre grava (sobrescreve a cassette). | Sim |
| `off` / `live` / `passthrough` | bate na API real, não grava nada. | Sim |
| `""` / `auto` (default do código) | grava se a cassette não existir, senão reproduz. | Sim, na 1ª vez |

## Rodando via `make` (recomendado)

```bash
make testacc-replay     # MGC_VCR_MODE=replay, endpoint/key/polling com defaults, timeout 30m
make testacc-record     # MGC_VCR_MODE=record, exige MGC_ENDPOINT+MGC_API_KEY, timeout 180m
make testacc            # roda os TestAcc* respeitando o MGC_VCR_MODE atual, timeout 180m
```

O `testacc-replay` já preenche valores inócuos quando você não passa nada:

```
MGC_ENDPOINT=https://replay.invalid
MGC_API_KEY=00000000-0000-4000-8000-000000000000
MGC_POLLING_INTERVAL=1ms
```

`MGC_POLLING_INTERVAL=1ms` faz o polling de status (`utils.PollingInterval`) não
dormir durante o replay — a suíte voa.

### Rodar só um subconjunto: `RUN`

Todos os alvos aceitam `RUN` (filtro `-run` do `go test`; default `TestAcc`):

```bash
make testacc-replay RUN=TestAccKubernetesCluster
make testacc-replay RUN='TestAccKubernetesCluster_basic'
make testacc-record RUN=TestAccKubernetesCluster_basic   # regrava só uma cassette
```

## Rodando via `go test` direto

Se quiser pular o `make`, os alvos são só açúcar sobre isto:

```bash
# Replay (equivalente ao make testacc-replay)
TF_ACC=1 MGC_VCR_MODE=replay \
  MGC_ENDPOINT=https://replay.invalid \
  MGC_API_KEY=00000000-0000-4000-8000-000000000000 \
  MGC_POLLING_INTERVAL=1ms \
  go test -v ./mgc/... -run TestAcc -timeout 30m

# Record
TF_ACC=1 MGC_VCR_MODE=record \
  MGC_ENDPOINT=https://sua-api MGC_API_KEY=<uuid> \
  go test -v ./mgc/... -run TestAcc -timeout 180m
```

Para depurar um único teste com log verboso:

```bash
TF_ACC=1 MGC_VCR_MODE=replay MGC_ENDPOINT=https://replay.invalid \
  MGC_API_KEY=00000000-0000-4000-8000-000000000000 MGC_POLLING_INTERVAL=1ms \
  TF_LOG=INFO go test -v ./mgc/kubernetes/... -run TestAccKubernetesCluster_basic
```

## Referência de variáveis de ambiente

| Variável | Para quê | Default |
|----------|----------|---------|
| `TF_ACC` | liga os acc tests (sem ela, skip) | — (os alvos `make` setam `1`) |
| `MGC_ENDPOINT` | URL raiz da API alvo | obrigatória em record/live; `https://replay.invalid` no replay |
| `MGC_API_KEY` | api_key do provider e dos SDK clients dos testes | obrigatória em record/live; UUID dummy no replay |
| `MGC_VCR_MODE` | modo do recorder | `auto` no código; alvos setam `replay`/`record` |
| `MGC_REGION` | região do provider e do sweeper | `br-se1` |
| `MGC_POLLING_INTERVAL` | intervalo de polling de status | `1ms` no replay |
| `RUN` (var do make) | filtro `-run` | `TestAcc` |
| `TF_LOG` | log do plugin-testing/terraform | — |

## Gravando cassettes novas (fluxo de record)

1. Aponte para uma API de verdade e grave:
   ```bash
   MGC_ENDPOINT=https://sua-api MGC_API_KEY=<uuid> make testacc-record RUN=TestAccMeuRecurso
   ```
2. As cassettes saem em `mgc/<service>/testdata/cassettes/`. Segredos
   (`Authorization`, `X-Api-Key`, `api_key`, `token`, `password`, ...) e headers
   voláteis (`Date`, `X-Request-Id`, ...) são **scrubbados** antes de salvar
   (`scrubHook` em `vcr.go`), então é seguro commitar.
3. **Confira o diff** e commit as cassettes junto com o código do teste.
4. Valide que reproduz limpo: `make testacc-replay RUN=TestAccMeuRecurso`.

> Nomes de recurso são determinísticos (seed = nome do teste), então record e
> replay geram exatamente os mesmos `tf-acctest-<kind>-<rand>` — o diff da
> cassette fica estável.

## Limpando recursos vazados (sweepers)

Se um record/live for interrompido, pode deixar recurso pago para trás. O sweeper
lista por SDK e deleta **somente** o que casa com o prefixo `tf-acctest-`:

```bash
# DESTRUTIVO: deleta recursos tf-acctest-* no MGC_ENDPOINT
MGC_ENDPOINT=https://sua-api MGC_API_KEY=<uuid> make sweep

# Restringir a sweepers específicos
... make sweep SWEEP_RUN=mgc_kubernetes_cluster
```

O sweeper **não** usa VCR (é limpeza real) e usa `MGC_REGION` (default `br-se1`).
Nunca roda no replay. Detalhes em `.claude/sweeper_memo.md`.

## Troubleshooting

| Sintoma | Causa provável / solução |
|---------|--------------------------|
| Testes `TestAcc*` são **skipados** | falta `TF_ACC=1` (use os alvos `make`) ou não há binário `terraform` no `PATH`. |
| `requested interaction not found` no replay | a cassette não existe ou ficou dessincronizada → regrave com `make testacc-record RUN=<teste>`. |
| `invalid MGC_VCR_MODE=...` | valor inválido; use `replay`, `record` ou `off`. |
| `MGC_ENDPOINT is required` | rodou record/live/sweep sem `MGC_ENDPOINT`/`MGC_API_KEY`. |
| Replay "trava" / lento | use `MGC_POLLING_INTERVAL=1ms` (o `make testacc-replay` já faz). |
| Matcher não casa após mudar a requisição | a request mudou (path/query/body) → regrave a cassette. |

## Onde estão as coisas

- Harness e helpers: `mgc/internal/acctest/acctest.go`
- Camada VCR (modos, matcher, scrub, nomes determinísticos): `mgc/internal/acctest/vcr.go`
- Referência de teste: `mgc/kubernetes/resource_cluster_generated_acc_test.go`
- Sweeper de referência: `mgc/kubernetes/resource_cluster_sweeper_test.go`
- Cassettes: `mgc/<service>/testdata/cassettes/*.yaml`
- Alvos: `make help` lista todos (`testacc`, `testacc-record`, `testacc-replay`, `sweep`).
