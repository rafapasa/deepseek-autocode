Guia de criação de issues para o ds-ac
Documento de referência. Consolida o aprendizado desta sessão. Vira o manual de como escrever issues que o ds-ac executa com segurança, sem retrabalho e sem varrer o projeto.

1. Causa raiz do erro do tenantID
A issue pedia:

GetByTelefone(ctx, tenantID uint, telefone string) e WHERE telefone = ? AND tenant_id = ?

O ds-ac implementou sem tenantID. Por quê?

Causa raiz: a descrição do item 1 (repo) é prosa longa com assinaturas embutidas no meio de 15 outras frases. O modelo lê, entende o "espírito" (GetByTelefone), e ao gerar o código simplifica — porque prosa não compila. Ele priorizou o nome do método e perdeu os parâmetros.

Fator agravante: o WHERE estava descrito em prosa (com Where("telefone = ? AND tenant_id = ?", telefone, tenantID)), não em bloco de código isolado. O modelo reorganizou.

Lição: o ds-ac respeita código, não prosa. O que está em bloco de código vira output fiel. O que está em texto corrido vira interpretação.

2. Regra de ouro
Toda especificação de assinatura, WHERE, struct ou interface vai em bloco de código compilável — nunca em prosa.

Não escreva:

"Adicionar GetByTelefone que recebe tenantID e telefone, e faz Where com telefone e tenant_id"

Escreva:

text
GetByTelefone(ctx context.Context, tenantID uint, telefone string) (*models.Cliente, error)

Implementação:
r.db.WithContext(ctx).
    Where("telefone = ? AND tenant_id = ?", telefone, tenantID).
    First(&cliente)
O modelo copia. O tenantID não some.

3. Estrutura da issue
Campos obrigatórios
json
{
  "demanda": "frase curta e direta, com o objetivo. Sem 'faça o que for necessário'.",
  "raiz": "/caminho/absoluto/do/projeto",
  "estrutura": [ ... tree do projeto ... ],
  "rules": [ ... regras globais ... ],
  "arquivos": [ ... lista fechada dos arquivos que podem ser tocados ... ],
  "tarefas": [ ... uma por arquivo, com descrição ... ]
}
Por que cada campo importa
Campo	Função	Erro se faltar
demanda	Objetivo, escopo	LLM inventa escopo
raiz	Diretório base	LLM varre /
estrutura	Árvore de arquivos	LLM faz list_dir a rodo, queima token
rules	Contrato global	LLM viola padrão (erros, mapper, response)
arquivos	Lista fechada	LLM toca arquivo errado
tarefas	O que fazer em cada arquivo	LLM decide sozinho, erra
Regra: se um campo não está preenchido, o LLM preenche por conta. Sempre.

4. demanda — como escrever
Formato: uma frase, verbo no infinitivo, escopo explícito.

Bom:

"Normalizar o model Cliente ao contrato de repositório e service. Fluxo handler → service → repo → service → handler com tipos corretos. Sem features novas, sem buildar, sem testar."

Ruim:

"Melhorar o cliente"
"Fazer o que for necessário para o cliente funcionar"
"Refatorar o cliente para o padrão do projeto"

Por quê: a LLM precisa de escopo fechado. "Melhorar" abre infinitas interpretações.

5. estrutura — sempre incluir
A árvore do projeto elimina as chamadas list_dir do ds-ac. Sem ela, o LLM:

Lista a raiz

Lista internal/

Lista internal/service/

Lista internal/repository/

... 10-15 turnos só de descoberta

Com ela, o LLM sabe exatamente onde cada arquivo está. Economia direta de token e de turnos.

Formato: o tree completo, com type directory/file. Não precisa de conteúdo, só nomes.

Dica: gerar com um script (tree -J ou equivalente) e colar. Manter atualizado antes de cada issue.

6. rules — o contrato
Regras globais que se aplicam a toda a issue. Curtas, imperativas, sem ambiguidade.

Modelo base (adapte por projeto)
text
Erros SEMPRE via apperror (apperror.NewNotFoundError, NewBadRequestError, ...). Nunca errors.New.
Conversoes DTO<->Model SEMPRE via gokit/mapper. Nunca copiar campo a campo.
Repostas HTTP SEMPRE via gokit/response (response.OK, response.Paginated, response.AppError). Nunca c.JSON direto.
Repo: interface final so com GetByID, GetBy<Unique>, FindWithFilters, Create, Update, Delete, WithTx.
Repo Create e Update recebem DTO e devolvem *models.X.
Repo buscas recebem parametros primitivos + filters, devolvem []models.X ou *models.X.
FindWithFilters aplica filter.ApplyFilters(query, models.X{}, filters). Slice vazio nunca e nil.
Service mantem TODOS os metodos de negocio existentes com os MESMOS nomes. Nao renomear, nao remover.
Service adiciona/renomeia APENAS os metodos do fluxo handler->repo.
Service GetBy* PROPAGA NotFound. Quem precisa engolir trata localmente com isNotFound.
Handler List usa response.GetPaginatedRequest(c), injeta filters["tenant_id"], chama service.List.
NAO buildar. NAO rodar testes. NAO rodar go fmt. Apenas editar os arquivos listados.
NAO criar DTOs novos. Usar os que ja existem.
NAO tocar em outros models.
Regras específicas de multi-tenant (crítico)
text
Campos unicos por tenant (telefone, cpf, cnpj, email) SO sao unicos em conjunto com tenant_id.
GetByTelefone(ctx, tenantID uint, telefone string) — sempre com tenantID.
WHERE telefone = ? AND tenant_id = ? — sempre com AND tenant_id.
Nunca buscar por telefone/cpf/cnpj/email sem tenant_id.
Essa regra precisa estar no topo das rules quando o model tem campos únicos por tenant. Foi o erro que aconteceu.

7. arquivos — lista fechada
Lista explícita dos arquivos que podem ser tocados. Nada fora dela.

Por quê: sem a lista, o LLM decide o que tocar. Com a lista, ele se limita.

Formato: paths relativos à raiz.

Exemplo:

json
"arquivos": [
  "internal/repository/cliente_repo.go",
  "internal/service/cliente_service.go",
  "internal/service/interface.go",
  "internal/server/api_handlers.go",
  "internal/mocks/cliente_repo_mock.go",
  "internal/mocks/cliente_service_mock.go"
]
Se a issue precisa de um arquivo que não está na lista, é sinal de que o escopo está errado — pare e reveja.

8. tarefas — uma por arquivo, com bloco de código
Essa é a parte mais importante. Cada tarefa:

id — número sequencial

arquivo — path exato

tipo — repo / service / interface / handler / mock / dto

descricao — o que fazer, com assinaturas em bloco de código

Anatomia de uma tarefa bem escrita
text
1) Objetivo (1 frase): "Reescrever a interface X conforme o contrato."

2) Interface final (bloco de código compilável):
```go
type ClienteRepositoryInterface interface {
    GetByID(ctx context.Context, id uint) (*models.Cliente, error)
    GetByTelefone(ctx context.Context, tenantID uint, telefone string) (*models.Cliente, error)
    // ...
}
Remover (lista explícita):
FindByID, FindByTelefone, FindByTenant, ...

Implementação (bloco de código, por método):

go
// GetByTelefone
r.db.WithContext(ctx).
    Where("telefone = ? AND tenant_id = ?", telefone, tenantID).
    First(&cliente)
Comportamento de erro (bloco de código ou regra):

go
if errors.Is(err, gorm.ErrRecordNotFound) {
    return nil, apperror.NewNotFoundError("cliente não encontrado")
}
return nil, apperror.NewInternalError("falha ao buscar", err)
text

**O que NUNCA fazer:**
- Deixar assinatura só em prosa
- Deixar `WHERE` só em prosa
- Dizer "siga o padrão do X" sem citar o padrão
- Dizer "faça o que for necessário"
- Dizer "ajuste os testes se necessário" (ou o teste entra no escopo, ou não)

---

## 9. Estrutura de uma tarefa — template
TAREFA <id> — <arquivo>

OBJETIVO:
<1-2 frases. Só o que deve ser feito. Sem contexto do projeto.>

INTERFACE FINAL (bloco de código compilável):
<assinaturas exatas>

REMOVER:
<lista explícita do que sai>

IMPLEMENTAÇÃO (bloco de código por método crítico):
<código>

ERROS:
<bloco de código ou regra curta>

NÃO TOCAR:
<o que fica intacto neste arquivo>

text

---

## 10. Erros comuns e como evitar

| Sintoma | Causa | Fix |
|---------|-------|-----|
| LLM omitiu parâmetro de assinatura | Assinatura só em prosa | Bloco de código |
| LLM usou `WHERE x = ?` sem `tenant_id` | Falta regra explícita multi-tenant | Adicionar nas `rules` e no bloco de código |
| LLM varreu 30 arquivos | `estrutura` ausente ou incompleta | Sempre incluir tree |
| LLM tocou arquivo fora do escopo | `arquivos` vazio ou `demanda` vaga | Lista fechada + demanda objetiva |
| LLM renomeou método que era pra ficar | Regra "manter X" ausente | Listar explicitamente o que mantém |
| LLM buildou sem pedir | Regra "NÃO buildar" ausente | Sempre incluir |
| LLM criou DTO novo | Regra "usar DTOs existentes" ausente | Sempre incluir |
| LLM alterou teste que não era pra tocar | Escopo do teste ambíguo | Decidir: entra ou não entra. Nunca "se necessário". |

---

## 11. Checklist antes de rodar

- [ ] `demanda` é uma frase objetiva, com verbo e escopo
- [ ] `raiz` é caminho absoluto, correto
- [ ] `estrutura` está atualizada (tree do projeto)
- [ ] `rules` inclui: apperror, mapper, response, contrato repo, contrato service, "NÃO buildar", "NÃO testar", "NÃO tocar em outros models", **regra multi-tenant** quando aplicável
- [ ] `arquivos` é lista fechada
- [ ] Cada `tarefa` tem: objetivo, interface final em bloco de código, remover (lista), implementação em bloco de código, comportamento de erro
- [ ] Assinaturas estão **compiláveis**, não em prosa
- [ ] `WHERE` crítico está em bloco de código com `AND tenant_id`
- [ ] Se o teste muda, ele está na lista de `arquivos`. Se não muda, há regra dizendo "não tocar"
- [ ] Nenhuma frase do tipo "faça o que for necessário", "ajuste se precisar", "siga o padrão"

---

## 12. Princípios finais

1. **Prosa é pedido, código é ordem.** O ds-ac obedece o que está em bloco de código.
2. **Escopo fechado > escopo aberto.** Lista de arquivos, lista do que remover, lista do que manter.
3. **A estrutura elimina a descoberta.** Sempre incluir o tree.
4. **Regra multi-tenant é obrigatória quando há campos únicos por tenant.** Sempre com `AND tenant_id`.
5. **O que não está escrito, a LLM inventa.** Se é crítico, escreve.
6. **Uma issue = um model.** Não misturar Cliente + Pedido + Produto na mesma issue.
7. **O teste decide-se: entra ou não entra.** Nunca "se necessário".
8. **Toda issue tem que ser executável sem o autor presente.** Se você precisa estar online pra responder dúvidas, a issue está incompleta.
9. **Se a issue ficou grande demais (>6 arquivos), divide.** Uma issue por arquivo-chave é melhor que uma issue com 15 arquivos.
10. **A estrutura da issue é o contrato.** Ela não é sugestão. É lei.

---

## 13. Template pronto (copiar e preencher)

```json
{
  "demanda": "<verbo + objeto + escopo. 1 frase.>",
  "raiz": "/caminho/absoluto",
  "estrutura": [ <tree> ],
  "rules": [
    "Erros SEMPRE via apperror.",
    "Conversoes DTO<->Model SEMPRE via gokit/mapper.",
    "Respostas HTTP SEMPRE via gokit/response.",
    "Repo: interface final so com GetByID, GetBy<Unique>, FindWithFilters, Create, Update, Delete, WithTx.",
    "Repo Create e Update recebem DTO e devolvem *models.X.",
    "Repo buscas recebem params + filters e devolvem Model(s).",
    "FindWithFilters aplica filter.ApplyFilters(query, models.X{}, filters).",
    "Service mantem metodos de negocio existentes com MESMOS nomes.",
    "Service adiciona/renomeia APENAS fluxo handler->repo.",
    "Service GetBy* PROPAGA NotFound.",
    "Handler List usa GetPaginatedRequest + filters[\"tenant_id\"] + response.Paginated.",
    "Campos unicos por tenant SO sao unicos com tenant_id. WHERE sempre com AND tenant_id.",
    "NAO buildar. NAO testar. NAO go fmt.",
    "NAO criar DTOs novos.",
    "NAO tocar em outros models."
  ],
  "arquivos": [ "<paths relativos>" ],
  "tarefas": [
    {
      "id": "1",
      "arquivo": "<path>",
      "tipo": "repo|service|interface|handler|mock|dto",
      "descricao": "<objetivo curto>\n\nINTERFACE FINAL:\n```go\n<assinaturas>\n```\n\nREMOVER:\n<lista>\n\nIMPLEMENTACAO:\n```go\n<codigo por metodo>\n```\n\nERROS:\n```go\n<trecho>\n```\n\nNAO TOCAR:\n<lista>"
    }
  ]
}
