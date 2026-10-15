CLAUDE.md — Especificação do Sistema de Controle Financeiro
1. Objetivo do projeto

Desenvolver uma aplicação completa de controle financeiro pessoal e doméstico, utilizando Go no backend e um banco de dados SQL relacional.

A aplicação deverá permitir que múltiplos usuários tenham contas independentes, registrem e administrem suas informações financeiras, acompanhem gastos mensais, controlem receitas e despesas e gerenciem seu patrimônio, incluindo investimentos.

O sistema deve ser implementado com foco em:

    Segurança e isolamento dos dados entre usuários.
    Persistência confiável de todas as informações.
    Integridade financeira e consistência dos cálculos.
    Arquitetura organizada, modular e sustentável.
    Interface intuitiva e responsiva.
    Regras de negócio explícitas e testáveis.
    Tratamento adequado de erros.
    Cobertura de testes automatizados.
    Facilidade de manutenção e evolução.

O objetivo é entregar uma aplicação funcional, completa e utilizável, não apenas um protótipo ou uma demonstração visual.
2. Escopo e restrições
2.1 Incluído no escopo

    Backend desenvolvido em Go.
    Banco de dados relacional SQL.
    Autenticação e gerenciamento de contas de usuários.
    Controle de receitas e despesas.
    Controle mensal de gastos domésticos.
    Gerenciamento de categorias.
    Gestão de contas financeiras.
    Gestão patrimonial.
    Cadastro e acompanhamento de investimentos.
    Dashboard financeiro.
    Relatórios e filtros.
    Edição, exclusão e consulta de registros próprios.
    Validação dos dados.
    Segurança da aplicação.
    Migrações do banco de dados.
    Testes unitários, de integração e de autorização.
    Documentação técnica e instruções para execução local.

2.2 Explicitamente fora do escopo

Não implementar nem configurar:

    Git ou estratégias de versionamento.
    Repositórios remotos.
    GitHub Actions ou outras ferramentas de CI/CD.
    Pipelines de integração ou entrega contínua.
    Infraestrutura de DevOps.
    Provisionamento de servidores.
    Orquestração de contêineres em produção.
    Fluxos de deploy automatizado.
    Configuração de cloud ou infraestrutura como código.

Não executar comandos Git nem criar arquivos de configuração relacionados a esses processos.

Docker Compose poderá ser utilizado exclusivamente como conveniência local para executar dependências, se necessário. Não desenvolver infraestrutura de produção.
3. Diretrizes para o desenvolvimento

Antes de implementar funcionalidades:

    Inspecionar os arquivos existentes no projeto.
    Identificar a estrutura, as tecnologias já adotadas e o que já está implementado.
    Preservar funcionalidades existentes que estejam corretas.
    Definir a arquitetura, o modelo de dados e as regras de negócio.
    Implementar o sistema em etapas funcionais.
    Executar os testes pertinentes após cada etapa.
    Corrigir falhas antes de considerar uma etapa concluída.
    Manter a documentação atualizada.

Não presumir que uma funcionalidade está concluída apenas porque sua interface existe.

Uma funcionalidade só é considerada implementada quando possui, conforme aplicável, interface, validação, regras de negócio, persistência, autorização, tratamento de erros e testes.

Não deixar botões sem ação, formulários que não persistam dados, telas com informações fictícias apresentadas como reais ou funcionalidades principais representadas apenas por placeholders.

Quando uma decisão técnica não estiver especificada, escolher a alternativa mais simples, segura e sustentável, documentando decisões relevantes.

Não interromper o desenvolvimento para solicitar aprovação de decisões técnicas rotineiras.
4. Stack tecnológica

Adotar preferencialmente as seguintes tecnologias:
Backend

    Go, utilizando uma versão estável e suportada.
    net/http para o servidor HTTP, podendo adotar roteamento adicional quando houver benefício concreto.
    database/sql para acesso relacional, com driver apropriado para o banco escolhido.
    PostgreSQL como banco de dados padrão.
    Consultas SQL parametrizadas.
    Migrações versionadas do esquema do banco.
    Contextos Go para cancelamento e limites de execução das operações.
    Logs estruturados.
    Configuração por variáveis de ambiente.

Frontend

Se o projeto ainda não possuir uma interface, desenvolver uma aplicação web responsiva.

Preferir uma arquitetura de frontend que consuma a API HTTP do backend e mantenha separação clara entre apresentação e regras de negócio.

A tecnologia de frontend pode ser escolhida considerando simplicidade, manutenção e compatibilidade com o projeto existente. Se necessário, utilizar TypeScript com React.

A interface deverá estar em português brasileiro (pt-BR), utilizar o real brasileiro (BRL) como moeda padrão e apresentar datas no formato brasileiro.
Banco de dados

Utilizar PostgreSQL como padrão, salvo se o projeto existente já possuir uma escolha justificada que deva ser preservada.

Não utilizar armazenamento em memória como fonte definitiva dos dados.

O banco de dados deverá ser a fonte persistente das informações financeiras.
Dependências

Evitar dependências desnecessárias. Preferir bibliotecas estáveis, mantidas e adequadas ao propósito.

Não criar abstrações complexas sem necessidade real.
5. Arquitetura da aplicação

Organizar o código por domínio e responsabilidade, evitando tanto um monólito desorganizado quanto uma fragmentação excessiva.

Uma estrutura inicial recomendada:

project/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── auth/
│   ├── user/
│   ├── dashboard/
│   ├── transaction/
│   ├── category/
│   ├── account/
│   ├── budget/
│   ├── household/
│   ├── asset/
│   ├── investment/
│   ├── report/
│   ├── database/
│   ├── middleware/
│   └── config/
├── migrations/
├── web/
├── tests/
├── .env.example
├── .gitignore
├── go.mod
└── README.md

A estrutura é uma referência, não uma exigência de criar pastas vazias. Adaptá-la ao projeto real.

Separar as responsabilidades:

    Handler: recebe requisições, valida o formato de entrada e produz respostas HTTP.
    Service: contém regras de negócio e coordena operações.
    Repository: executa operações de persistência.
    Domain: representa conceitos, invariantes e regras do negócio.
    Middleware: trata autenticação, autorização, limites e aspectos transversais.
    Frontend: apresenta informações e interage com a API.

Regras financeiras não devem existir exclusivamente no frontend.

Operações que envolvam múltiplas alterações relacionadas devem utilizar transações SQL.

Evitar dependência circular entre módulos.
6. Regra fundamental: isolamento entre usuários

Esta é uma das regras mais importantes de todo o sistema.

Cada usuário deverá ter acesso exclusivamente aos próprios dados financeiros, sem interferência ou exposição dos dados de outros usuários.
6.1 Identificação do usuário

    Cada conta deverá possuir um identificador único e estável.
    A identidade do usuário autenticado deverá ser obtida exclusivamente de uma sessão ou credencial validada pelo backend.
    Nunca confiar em user_id enviado pelo frontend como prova de identidade ou autorização.
    Toda operação protegida deverá verificar a identidade autenticada.

6.2 Autorização em cada operação

Toda consulta, criação, alteração e exclusão de dados privados deverá respeitar o usuário autenticado.

Exemplos:

    Um usuário não poderá consultar uma transação de outro usuário alterando o ID na URL.
    Um usuário não poderá editar uma categoria de outra conta.
    Um usuário não poderá excluir um investimento pertencente a outro usuário.
    Um usuário não poderá consultar relatórios de outro usuário modificando parâmetros da requisição.
    Um usuário não poderá alterar o identificador do proprietário enviando um JSON manipulado.
    O dashboard deverá considerar exclusivamente os dados autorizados.
    Filtros, paginação, exportações e operações em lote deverão respeitar a mesma regra.

A autorização deverá ser aplicada no backend, preferencialmente nas operações de serviço e nas consultas ao banco.

Não carregar um registro globalmente e confiar apenas na interface para impedir sua edição.

Sempre que possível, realizar consultas com o identificador do recurso e o proprietário autenticado na mesma operação SQL.

Exemplo conceitual:

SELECT id, description, amount, type
FROM transactions
WHERE id = $1
  AND user_id = $2;

O parâmetro $2 deve vir da identidade autenticada, nunca de um valor arbitrário fornecido pelo cliente.
6.3 Integridade relacional

Garantir também que relacionamentos entre entidades pertençam ao mesmo usuário.

Exemplos:

    Uma transação não poderá referenciar uma conta de outro usuário.
    Uma transação não poderá utilizar uma categoria privada de outro usuário.
    Uma movimentação de investimento não poderá referenciar uma posição de investimento alheia.
    Uma meta ou orçamento não poderá associar recursos de contas diferentes.

Implementar as verificações na camada de serviço e, quando apropriado, reforçá-las com chaves estrangeiras e restrições compostas no banco.

Avaliar o uso de Row-Level Security (RLS) do PostgreSQL como defesa adicional, sem substituir a autorização da aplicação. Se adotada, configurar corretamente o contexto de usuário em cada transação e impedir vazamentos de contexto entre conexões reutilizadas.
6.4 Testes obrigatórios de isolamento

Criar pelo menos dois usuários independentes nos testes.

Verificar que:

    O usuário A não consegue ler dados do usuário B.
    O usuário A não consegue editar dados do usuário B.
    O usuário A não consegue excluir dados do usuário B.
    O usuário A não consegue associar recursos do usuário B aos próprios registros.
    Os dashboards de A e B apresentam resultados independentes.
    As exportações de A não contêm dados de B.
    A tentativa de acesso indevido não modifica nenhum registro.
    Operações em lote não ultrapassam o limite de propriedade do usuário.

Esses testes são obrigatórios para considerar o sistema seguro.
7. Autenticação e gerenciamento de usuários

Implementar um fluxo completo de gerenciamento de conta.
Funcionalidades

    Cadastro de usuário.
    Login.
    Logout.
    Consulta e edição do perfil.
    Alteração de senha.
    Recuperação de senha, quando houver um mecanismo de entrega de mensagens configurado.
    Encerramento seguro da sessão.
    Validação de e-mail.
    Proteção contra tentativas excessivas de autenticação.

Segurança

    Nunca armazenar senhas em texto puro.
    Utilizar algoritmo apropriado para hashing de senhas, preferencialmente Argon2id ou uma alternativa robusta.
    Nunca registrar senhas, tokens ou credenciais nos logs.
    Utilizar sessões seguras ou tokens com expiração e estratégia de revogação bem definidas.
    Para uma aplicação web tradicional, preferir cookies HttpOnly, Secure em HTTPS e política SameSite adequada.
    Proteger operações autenticadas contra CSRF quando aplicável.
    Validar rigorosamente os dados de entrada.
    Evitar enumerar contas por meio de mensagens de erro excessivamente específicas.
    Não devolver hashes de senha ou dados internos desnecessários nas respostas.
    Não implementar autenticação fictícia ou um usuário fixo compartilhado.

A funcionalidade de recuperação de senha não deverá aparentar estar operacional se o mecanismo de envio de mensagens não estiver configurado.
8. Domínios funcionais da aplicação

A aplicação deverá possuir navegação clara, com pelo menos as seguintes áreas:

    Dashboard.
    Gastos mensais.
    Receitas.
    Contas financeiras.
    Patrimônio.
    Investimentos.
    Categorias.
    Relatórios.
    Perfil e configurações.

As telas deverão utilizar dados persistidos e respeitar a identidade do usuário autenticado.
9. Dashboard financeiro

Criar uma tela inicial com visão consolidada da situação financeira.
Indicadores

    Saldo total das contas financeiras.
    Receitas do mês selecionado.
    Despesas do mês selecionado.
    Resultado mensal.
    Patrimônio líquido.
    Total investido.
    Distribuição de despesas por categoria.
    Evolução mensal das receitas e despesas.
    Últimas movimentações.

Comportamento

    Permitir selecionar o período.
    Exibir estados de carregamento, erro e ausência de dados.
    Atualizar os indicadores após operações que afetem os resultados.
    Não somar o patrimônio duas vezes por conta de investimentos que já estejam representados em contas financeiras.
    Diferenciar fluxo financeiro, saldo de contas, ativos, passivos e patrimônio líquido.
    Apresentar claramente a data de referência de valores patrimoniais.

O dashboard deve calcular os indicadores a partir das informações persistidas, sem valores fictícios.
10. Aba de gastos mensais

Esta é uma das funcionalidades centrais da aplicação.

O objetivo é permitir que o usuário controle as despesas pessoais e domésticas mês a mês.
10.1 Lançamentos financeiros

Cada transação deverá possuir, conforme aplicável:

    Identificador.
    Proprietário.
    Descrição.
    Valor.
    Tipo: receita ou despesa.
    Data da movimentação.
    Categoria.
    Conta financeira.
    Observações.
    Status de pagamento ou recebimento, quando aplicável.
    Data de criação.
    Data da última atualização.

Permitir criar, consultar, editar e excluir os lançamentos próprios.

Validar os campos obrigatórios e rejeitar valores inválidos.
10.2 Controle mensal

O usuário deverá poder selecionar mês e ano para consultar os dados correspondentes.

A tela deverá apresentar:

    Total de receitas no período.
    Total de despesas no período.
    Resultado do período.
    Despesas agrupadas por categoria.
    Lista de lançamentos.
    Despesas pagas e pendentes, quando aplicável.
    Comparativo com outros períodos.
    Filtros por categoria, conta, tipo e intervalo de datas.

Os totais deverão considerar apenas os registros pertinentes ao período selecionado e a regra financeira adotada.
10.3 Categorias

Permitir criar, editar e excluir categorias próprias.

Exemplos de categorias de despesas:

    Moradia.
    Água e energia.
    Internet e telefone.
    Alimentação.
    Transporte.
    Saúde.
    Educação.
    Lazer.
    Assinaturas.
    Compras pessoais.
    Impostos.
    Outros.

Exemplos de categorias de receitas:

    Salário.
    Trabalho autônomo.
    Rendimentos.
    Aluguel recebido.
    Outras receitas.

As categorias sugeridas podem ser oferecidas como padrão, mas o usuário deve poder personalizar as próprias categorias.

Categorias utilizadas em lançamentos históricos não devem ser excluídas de forma que cause perda de integridade ou informação. Oferecer arquivamento ou impedir a exclusão enquanto existirem referências, conforme a política definida.

Categorias globais de sistema, se existirem, devem ser separadas das categorias privadas e não podem ser alteradas indevidamente.
10.4 Despesas recorrentes

Implementar suporte a despesas recorrentes, permitindo definir:

    Descrição.
    Valor previsto.
    Categoria.
    Periodicidade.
    Data inicial.
    Data final opcional.
    Dia previsto para vencimento.
    Conta financeira, quando aplicável.
    Ativação e desativação.

A recorrência deve gerar lançamentos identificáveis e sem duplicação.

Uma ocorrência prevista não deverá ser confundida automaticamente com uma despesa efetivamente paga.

O sistema deverá permitir editar uma ocorrência ou alterar a regra de recorrência, distinguindo os efeitos futuros dos registros históricos.
10.5 Orçamento mensal

Permitir definir limites de gastos por categoria e período.

Exibir:

    Valor planejado.
    Valor efetivamente gasto.
    Saldo disponível no orçamento.
    Percentual consumido.
    Categorias que ultrapassaram o limite.

O orçamento deve ser individual e editável.

Alterações no orçamento não podem modificar retroativamente os lançamentos financeiros.
10.6 Consistência das despesas

Distinguir claramente:

    Despesa planejada.
    Despesa pendente.
    Despesa paga.
    Despesa cancelada.

A implementação deverá definir explicitamente quais estados entram nos relatórios de despesas realizadas, compromissos futuros e fluxo de caixa.

Uma despesa pendente não deverá ser contabilizada como pagamento efetivado sem uma regra explícita.
11. Contas financeiras

Implementar o gerenciamento de contas usadas para organizar o dinheiro do usuário.

Exemplos:

    Carteira.
    Conta corrente.
    Conta poupança.
    Conta digital.
    Conta de investimento.
    Outras contas configuráveis.

Cada conta deverá possuir:

    Identificador.
    Proprietário.
    Nome.
    Tipo.
    Instituição opcional.
    Moeda.
    Saldo inicial.
    Data de referência do saldo inicial.
    Status ativo ou arquivado.
    Datas de criação e atualização.

Permitir cadastrar, editar, consultar e arquivar contas próprias.
Regras

    O saldo atual deverá ser calculado de maneira consistente a partir do saldo inicial e das movimentações efetivadas.
    Transferências entre contas próprias não constituem receita nem despesa consolidada.
    Uma transferência deve gerar movimentações vinculadas, debitando a origem e creditando o destino.
    Os dois lados da transferência devem ser registrados atomicamente.
    Uma transferência não pode ter origem e destino inválidos ou pertencer a usuários diferentes.
    Editar ou excluir uma conta não pode apagar silenciosamente seu histórico financeiro.
    Alterações no saldo inicial devem ter semântica explícita e preservar a consistência histórica.

Caso existam moedas diferentes, não somar valores de moedas distintas sem conversão cambial válida e data de referência documentada.
12. Aba de patrimônio

O patrimônio deverá ser tratado como um domínio próprio, diferente do controle mensal de receitas e despesas.

O objetivo é permitir que o usuário acompanhe os bens, ativos, obrigações e a evolução do próprio patrimônio.
12.1 Visão patrimonial

Exibir:

    Total de ativos.
    Total de passivos.
    Patrimônio líquido.
    Total investido.
    Distribuição dos ativos por tipo.
    Evolução do patrimônio ao longo do tempo.
    Data da última atualização dos valores.
    Histórico de avaliações, quando implementado.

Definições:

Patrimônio bruto: soma dos valores dos ativos considerados na visão patrimonial.

Passivos: obrigações financeiras registradas, conforme a política de contabilização.

Patrimônio líquido: patrimônio bruto menos os passivos.

Os cálculos deverão seguir uma política consistente para evitar duplicação de valores.
12.2 Ativos

Permitir cadastrar e editar bens como:

    Imóveis.
    Veículos.
    Participações empresariais.
    Dinheiro em espécie, se não estiver representado em uma conta financeira.
    Outros bens patrimoniais.

Cada ativo deverá possuir:

    Nome ou descrição.
    Tipo.
    Valor atual estimado.
    Data de referência da avaliação.
    Observações.
    Status.
    Datas de criação e atualização.

Quando necessário, manter histórico de avaliações para permitir acompanhar a evolução patrimonial.

Editar o valor atual não deverá apagar automaticamente as avaliações históricas.
12.3 Passivos e obrigações

Permitir cadastrar obrigações como:

    Financiamentos.
    Empréstimos.
    Dívidas pessoais.
    Outras obrigações financeiras.

Registrar saldo devedor atual, data de referência, descrição e demais informações relevantes.

Não somar o valor original de uma dívida ao saldo devedor atual. O cálculo patrimonial deve considerar a obrigação no estado correto da data de referência.

A baixa de um passivo deve preservar o histórico.
12.4 Independência dos registros patrimoniais

O usuário deverá poder atualizar manualmente os valores dos ativos e passivos.

A aplicação não deverá afirmar que possui cotação ou avaliação de mercado atualizada quando o valor foi informado manualmente.

Mostrar a data de referência do valor e, quando aplicável, indicar que se trata de uma estimativa do usuário.
13. Aba de investimentos

A área de investimentos deve permitir cadastrar, editar e acompanhar investimentos pertencentes ao usuário.

A primeira versão deve funcionar sem depender de integração externa com corretoras ou instituições financeiras.
13.1 Cadastro de investimentos

Permitir cadastrar investimentos como:

    Renda fixa.
    Tesouro Direto.
    CDB, LCI e LCA.
    Fundos de investimento.
    Ações.
    ETFs.
    Fundos imobiliários.
    Previdência privada.
    Criptoativos.
    Outros ativos configuráveis.

Essas categorias são exemplos e não devem ser tratadas como uma lista fechada.

Cada investimento poderá possuir:

    Identificador.
    Proprietário.
    Nome.
    Tipo de ativo.
    Instituição ou corretora.
    Código ou ticker opcional.
    Quantidade de unidades, quando aplicável.
    Preço médio de aquisição, quando aplicável.
    Valor investido.
    Valor de mercado atual.
    Data da posição.
    Data de aquisição, quando aplicável.
    Vencimento, quando aplicável.
    Observações.
    Status.
    Datas de criação e atualização.

Os campos deverão variar conforme o tipo de investimento, sem exigir quantidade ou vencimento em ativos para os quais esses dados não façam sentido.
13.2 Operações e movimentações

Quando houver compra, venda, aporte, resgate, rendimento ou taxa, permitir registrar movimentações compatíveis com o tipo do ativo.

Cada movimentação deverá ter:

    Tipo.
    Data.
    Valor.
    Quantidade, quando aplicável.
    Custos ou taxas, quando aplicáveis.
    Referência ao investimento.
    Observações.

Movimentações que afetem contas financeiras e posições de investimentos devem ser transacionais, preservando a consistência entre os domínios.

Não implementar a mesma movimentação como uma despesa e como uma redução patrimonial adicional, gerando dupla contabilização.
13.3 Métricas de investimentos

Exibir, quando os dados disponíveis permitirem:

    Valor total investido.
    Valor de mercado.
    Ganho ou perda estimada.
    Rentabilidade simples.
    Distribuição por classe de ativo.
    Evolução da carteira.
    Histórico de aportes e resgates.

Definir claramente as fórmulas e os dados necessários para cada indicador.

Não apresentar uma rentabilidade precisa quando faltarem dados suficientes para calculá-la corretamente.

Distinguir resultados realizados de ganhos e perdas não realizados.

Não tratar aportes como lucro e resgates como prejuízo.

Não inventar cotações ou taxas de mercado. Quando não houver integração externa, utilizar os valores informados pelo usuário e identificar sua data de referência.
13.4 Patrimônio e investimentos

Um investimento pode fazer parte do patrimônio do usuário.

A mesma posição não pode ser somada duas vezes ao calcular o patrimônio total.

Definir uma fonte de verdade para cada posição e deixar explícito se o valor está incluído na carteira de investimentos, nas contas financeiras ou nos ativos patrimoniais.

Transferir dinheiro de uma conta para uma posição de investimento deve ser tratado como movimentação entre componentes patrimoniais, não como uma nova despesa de consumo.
14. Regras financeiras gerais
14.1 Valores monetários

    Nunca utilizar float32 ou float64 como fonte de verdade para cálculos monetários.
    Utilizar tipos decimais adequados no banco, como NUMERIC.
    Utilizar representação decimal segura no Go, com uma biblioteca apropriada ou estratégia equivalente.
    Definir e validar a precisão permitida.
    Arredondar somente conforme regras explícitas.
    Utilizar parâmetros SQL.
    Validar valores negativos, nulos, excessivos ou incompatíveis com a operação.

14.2 Datas e períodos

    Armazenar datas de movimentações financeiras como datas, quando não dependerem de horário.
    Armazenar instantes de auditoria em UTC.
    Definir explicitamente o fuso horário de apresentação.
    Considerar corretamente os limites dos meses e anos.
    Evitar erros de um dia causados por conversão de fuso.
    Definir como tratar lançamentos futuros, vencimentos e competências mensais.

14.3 Saldo, fluxo e patrimônio

Não confundir:

    Receita com entrada de dinheiro decorrente de transferência própria.
    Despesa com transferência para investimento.
    Saldo bancário com patrimônio líquido.
    Aporte com lucro.
    Valor de mercado com valor originalmente investido.
    Despesa pendente com despesa paga.
    Patrimônio total com soma de todos os módulos sem regras de deduplicação.

Os cálculos deverão ser centralizados e cobertos por testes.
14.4 Exclusão e histórico

Antes de excluir um registro, avaliar seus relacionamentos e seus efeitos financeiros.

Para registros que façam parte de um histórico financeiro relevante, preferir arquivamento, cancelamento ou estorno quando a exclusão física comprometer a rastreabilidade.

Não permitir que uma exclusão deixe saldos, transferências ou investimentos inconsistentes.
15. Persistência e integridade dos dados

A persistência é um requisito obrigatório e não negociável.
15.1 Banco de dados como fonte de verdade

Todas as informações de negócio deverão ser persistidas no banco de dados.

Isso inclui:

    Usuários.
    Preferências.
    Categorias.
    Contas financeiras.
    Transações.
    Orçamentos.
    Regras de recorrência.
    Ativos.
    Passivos.
    Investimentos.
    Movimentações.
    Avaliações patrimoniais.
    Configurações financeiras próprias de cada usuário.

Não utilizar variáveis globais, mapas em memória, arrays do frontend, localStorage ou dados mockados como armazenamento definitivo.

O frontend pode utilizar estado temporário para melhorar a experiência, mas o banco de dados deverá continuar sendo a fonte de verdade.
15.2 Comportamento após reinicialização

Após reiniciar o backend, recarregar a página ou autenticar-se novamente, os dados deverão continuar disponíveis.

Os registros não poderão desaparecer por reinicialização do processo ou atualização da interface.
15.3 Transações SQL

Utilizar transações para operações que envolvam várias alterações relacionadas.

Exemplos:

    Transferência entre contas.
    Registro de aporte associado a uma movimentação financeira.
    Venda ou resgate que altere posição e saldo.
    Exclusão lógica ou estorno com múltiplas consequências.
    Operações que atualizem mais de uma entidade financeira.

Se uma etapa falhar, todas as alterações relacionadas deverão ser revertidas.
15.4 Concorrência

Considerar requisições simultâneas e múltiplas sessões do mesmo usuário.

Utilizar restrições do banco, transações e estratégias de concorrência adequadas para impedir:

    Duplicação de operações.
    Atualizações perdidas.
    Saldos inconsistentes.
    Criação de transferências pela metade.
    Execução duplicada de recorrências.

Para operações financeiras que possam ser repetidas por falha de rede, implementar idempotência quando apropriado.
15.5 Migrações

Criar migrações explícitas para todas as alterações do esquema.

As migrações devem ser reproduzíveis em um banco vazio e preservar os dados existentes quando houver atualização de esquema.

Não depender de alterações manuais no banco para fazer a aplicação funcionar.
15.6 Restrições

Utilizar, conforme apropriado:

    Chaves primárias.
    Chaves estrangeiras.
    Restrições NOT NULL.
    Restrições UNIQUE.
    Restrições CHECK.
    Índices para consultas frequentes.
    Integridade referencial.
    Transações.

Não confiar exclusivamente na validação do frontend.
15.7 Exclusão de conta

Caso seja implementado o encerramento de conta, definir claramente quais dados serão excluídos, anonimizados ou mantidos por obrigação legal.

A operação deverá ser autenticada, confirmada e executada de maneira segura, sem deixar dados órfãos ou informações acessíveis por outros usuários.
16. Modelo de dados

Criar um esquema relacional coerente com os domínios acima.

As entidades abaixo são uma referência inicial:

    users
    user_preferences
    categories
    financial_accounts
    transactions
    transfers, se necessário
    recurring_rules
    budgets
    budget_categories, se necessário
    assets
    asset_valuations
    liabilities
    investments
    investment_transactions
    investment_valuations
    sessions, caso sejam utilizadas sessões persistidas
    audit_events, quando necessário

Não é obrigatório criar uma tabela para cada item se uma modelagem mais simples atender aos requisitos sem perder integridade.
Requisitos do modelo

    Identificadores únicos.
    Proprietário explícito para os dados privados.
    Relacionamentos consistentes.
    Campos de auditoria.
    Índices apropriados.
    Restrições financeiras.
    Políticas claras de exclusão.
    Tipos de dados corretos.
    Integridade entre usuário e recurso relacionado.

Quando uma entidade pertencer a um usuário, sua autorização deverá continuar válida em todas as consultas e operações relacionadas.

Documentar o modelo e as decisões de contabilização adotadas.
17. API HTTP

Criar uma API consistente, com rotas organizadas por domínio.

Exemplos de rotas:
Autenticação

    POST /api/v1/auth/register
    POST /api/v1/auth/login
    POST /api/v1/auth/logout
    POST /api/v1/auth/password/forgot
    POST /api/v1/auth/password/reset

Implementar as rotas de recuperação somente quando o mecanismo de entrega e validação estiver adequadamente configurado.
Perfil

    GET /api/v1/me
    PATCH /api/v1/me
    PATCH /api/v1/me/password

Dashboard

    GET /api/v1/dashboard/summary
    GET /api/v1/dashboard/monthly-flow
    GET /api/v1/dashboard/expenses-by-category

Transações

    GET /api/v1/transactions
    POST /api/v1/transactions
    GET /api/v1/transactions/{id}
    PATCH /api/v1/transactions/{id}
    DELETE /api/v1/transactions/{id}

Categorias

    GET /api/v1/categories
    POST /api/v1/categories
    PATCH /api/v1/categories/{id}
    DELETE /api/v1/categories/{id}

Contas

    GET /api/v1/accounts
    POST /api/v1/accounts
    GET /api/v1/accounts/{id}
    PATCH /api/v1/accounts/{id}
    DELETE /api/v1/accounts/{id}

A exclusão poderá ser substituída por arquivamento quando houver histórico relacionado.
Orçamentos

    GET /api/v1/budgets
    POST /api/v1/budgets
    PATCH /api/v1/budgets/{id}
    DELETE /api/v1/budgets/{id}

Patrimônio

    GET /api/v1/assets
    POST /api/v1/assets
    PATCH /api/v1/assets/{id}
    DELETE /api/v1/assets/{id}
    GET /api/v1/liabilities
    POST /api/v1/liabilities
    PATCH /api/v1/liabilities/{id}
    DELETE /api/v1/liabilities/{id}
    GET /api/v1/net-worth

Investimentos

    GET /api/v1/investments
    POST /api/v1/investments
    GET /api/v1/investments/{id}
    PATCH /api/v1/investments/{id}
    DELETE /api/v1/investments/{id}
    GET /api/v1/investments/summary
    GET /api/v1/investments/{id}/transactions
    POST /api/v1/investments/{id}/transactions

Adaptar as rotas à arquitetura escolhida e documentar os endpoints efetivamente implementados.
Padrões da API

    Utilizar códigos HTTP apropriados.
    Validar parâmetros, corpo e filtros.
    Definir formatos de resposta consistentes.
    Implementar paginação nas listas.
    Utilizar filtros explícitos.
    Não expor detalhes internos de SQL nas mensagens de erro.
    Não retornar dados sensíveis.
    Documentar a API.
    Garantir autorização em todas as rotas protegidas.

Para recursos inexistentes ou não autorizados, adotar uma política de resposta consistente que não revele informações privadas sobre a existência de recursos de terceiros.
18. Interface e experiência do usuário

A aplicação deverá ser visualmente consistente, simples de compreender e funcional.
Navegação

Utilizar uma navegação clara para:

    Dashboard.
    Gastos mensais.
    Receitas.
    Contas.
    Patrimônio.
    Investimentos.
    Relatórios.
    Configurações.

Requisitos visuais

    Interface responsiva para desktop e dispositivos móveis.
    Valores monetários formatados em BRL.
    Datas apresentadas no padrão brasileiro.
    Formulários com validação.
    Mensagens de sucesso e erro.
    Confirmação de operações destrutivas.
    Indicadores de carregamento.
    Estados vazios com orientação para o usuário.
    Paginação ou carregamento eficiente de listas extensas.
    Acessibilidade básica em formulários, navegação e componentes.
    Contraste adequado e hierarquia visual consistente.

Requisitos funcionais

    Os formulários devem persistir os dados por meio da API.
    Os dados devem ser recarregados do backend.
    Após uma operação, as telas afetadas devem refletir o resultado persistido.
    Erros de validação devem indicar o campo relevante.
    Erros do servidor devem ser tratados sem expor informações técnicas sensíveis.
    O usuário deverá conseguir editar os próprios dados sem depender de manipulação direta do banco.

Não utilizar informações fictícias no dashboard de produção.

Se houver dados de demonstração nos testes, mantê-los claramente separados da execução normal da aplicação.
19. Segurança adicional

Implementar medidas de segurança proporcionais a uma aplicação que manipula dados financeiros pessoais.

    Validar todas as entradas.
    Utilizar consultas SQL parametrizadas.
    Configurar limites para corpo de requisições.
    Validar e normalizar dados conforme as regras de negócio.
    Configurar CORS com origens permitidas explicitamente.
    Implementar limites de tentativas de login.
    Configurar timeouts HTTP e de acesso ao banco.
    Evitar vazamento de stack traces para o cliente.
    Não registrar dados financeiros completos ou credenciais sem necessidade.
    Proteger sessões e tokens.
    Utilizar cabeçalhos de segurança apropriados.
    Manter dependências razoavelmente atualizadas.
    Restringir operações de administração.
    Garantir que nenhuma rota privada possa ser acessada sem autenticação.
    Garantir que nenhuma operação de escrita possa ignorar a autorização.
    Aplicar políticas de retenção e minimização de dados quando pertinente.

Se houver cookies de autenticação, avaliar CSRF. Se houver conteúdo renderizado fornecido pelo usuário, proteger contra XSS.

Considerar os princípios da LGPD, incluindo finalidade, minimização, transparência, segurança e atendimento aos direitos dos titulares.

Não afirmar conformidade legal completa sem uma avaliação específica.
20. Auditoria e rastreabilidade

Implementar campos de criação e atualização para as entidades relevantes.

Quando apropriado, manter eventos de auditoria para ações importantes, como:

    Alteração de informações financeiras.
    Exclusão ou cancelamento de registros.
    Alteração de saldo inicial.
    Modificações patrimoniais.
    Operações relevantes em investimentos.
    Alterações de segurança da conta.

Os eventos deverão identificar o usuário responsável, a ação, a entidade afetada e a data.

Não registrar senhas, tokens ou informações de autenticação nos eventos.

A auditoria não deve substituir os dados financeiros principais nem ser usada como desculpa para permitir operações inconsistentes.
21. Relatórios e consultas

Implementar consultas que permitam ao usuário compreender a própria situação financeira.

Relatórios prioritários:

    Receitas e despesas por mês.
    Despesas por categoria.
    Evolução do saldo.
    Comparação entre meses.
    Orçamento versus gasto realizado.
    Contas e respectivos saldos.
    Composição patrimonial.
    Evolução do patrimônio líquido.
    Composição da carteira de investimentos.
    Histórico de movimentações.

Os filtros devem respeitar o proprietário autenticado e os limites de período definidos.

Utilizar consultas eficientes e índices apropriados.

Evitar buscar todos os registros para calcular métricas simples no frontend.

Garantir que os valores agregados correspondam aos dados detalhados apresentados ao usuário.
22. Testes obrigatórios

Não considerar a implementação concluída sem testes relevantes.
22.1 Testes unitários

Cobrir:

    Validação de valores.
    Cálculo de saldo.
    Cálculo de receitas e despesas.
    Regras de recorrência.
    Limites mensais.
    Cálculo de patrimônio líquido.
    Cálculo de métricas de investimentos.
    Validação de transações financeiras.
    Tratamento de datas e períodos.

22.2 Testes de integração

Executar contra um banco PostgreSQL de teste.

Cobrir:

    Migrações.
    Cadastro e consulta de dados.
    Edição e exclusão.
    Relacionamentos.
    Transações SQL.
    Rollback após falha.
    Restrições do banco.
    Persistência após reconexão.
    Operações concorrentes relevantes.

Não substituir os testes de integração por mocks que não validem o comportamento real do banco.
22.3 Testes de API

Cobrir:

    Autenticação.
    Autorização.
    Validação de entrada.
    Códigos HTTP.
    Paginação e filtros.
    Respostas de erro.
    Criação, edição e exclusão.
    Isolamento entre usuários.

22.4 Testes financeiros

Criar casos de teste explícitos para:

    Uma receita aumentar corretamente o saldo.
    Uma despesa diminuir corretamente o saldo.
    Uma transferência não alterar o resultado consolidado de receitas e despesas.
    Uma transferência atualizar as duas contas atomicamente.
    Um aporte não ser contabilizado como despesa de consumo no consolidado.
    Uma venda ou resgate não gerar dupla contabilização.
    Uma alteração patrimonial refletir-se corretamente no patrimônio líquido.
    Uma despesa pendente não ser confundida com uma despesa paga.
    A exclusão ou o estorno de uma operação preservar a consistência.
    Valores com casas decimais serem calculados corretamente.

22.5 Testes de segurança

Verificar:

    Acesso indevido por alteração de IDs.
    Tentativas de atribuição de outro proprietário.
    Manipulação de relacionamentos.
    Acesso sem autenticação.
    Escalada de privilégios.
    Vazamento de dados entre usuários.
    Requisições malformadas.
    Tentativas de SQL injection.
    Uso indevido de sessões.
    Repetição de operações financeiras sensíveis.

Executar os testes disponíveis e registrar de forma objetiva o que foi validado e o que ainda depende de configuração externa.
23. Desempenho e qualidade

Implementar:

    Paginação.
    Índices apropriados.
    Consultas SQL eficientes.
    Timeouts.
    Tratamento de erros.
    Pool de conexões configurado adequadamente.
    Logs estruturados.
    Encerramento controlado do servidor.
    Limites razoáveis de entrada.
    Separação de responsabilidades.
    Código idiomático em Go.

Evitar:

    Consultas N+1.
    Operações SQL sem parâmetros.
    Cálculos financeiros duplicados em diferentes camadas.
    Tratamento silencioso de erros.
    Uso de panic para erros de negócio esperados.
    Funções excessivamente grandes.
    Dependências desnecessárias.
    Complexidade arquitetural sem benefício.
    Comentários que apenas repetem o código.

Não sacrificar integridade financeira em favor de otimizações prematuras.
24. Configuração e execução local

A aplicação deverá poder ser executada localmente seguindo instruções claras.

Criar:

    Arquivo .env.example com nomes de variáveis e valores demonstrativos não sensíveis.
    Configuração de conexão com PostgreSQL.
    Instruções para criar ou disponibilizar o banco local.
    Comandos para executar as migrações.
    Comandos para iniciar o backend.
    Instruções para iniciar o frontend, se houver.
    Instruções para executar os testes.
    Instruções para verificar se a aplicação está funcionando.

Não incluir senhas reais ou credenciais privadas nos arquivos.

Validar as variáveis de ambiente obrigatórias na inicialização e apresentar mensagens de erro claras.

Não criar processos de deploy ou pipelines.
25. Documentação técnica

Criar um README que explique:

    Objetivo do sistema.
    Tecnologias utilizadas.
    Pré-requisitos.
    Estrutura do projeto.
    Configuração local.
    Configuração do banco.
    Execução das migrações.
    Inicialização do backend e frontend.
    Execução dos testes.
    Variáveis de ambiente.
    Endpoints principais.
    Decisões financeiras relevantes.

Documentar também as fórmulas e convenções utilizadas para:

    Saldo de contas.
    Receitas e despesas.
    Transferências.
    Orçamentos.
    Patrimônio líquido.
    Avaliação de investimentos.
    Ganhos realizados e não realizados.

Não documentar funcionalidades como concluídas quando estiverem incompletas.
26. Ordem de implementação

Desenvolver o projeto em etapas, sempre entregando funcionalidades integradas.
Etapa 1 — Fundação

    Inspecionar o projeto.
    Definir arquitetura.
    Configurar Go e PostgreSQL.
    Criar migrações iniciais.
    Implementar conexão com o banco.
    Criar configuração e tratamento de erros.
    Estabelecer padrões de código e testes.

Critério de conclusão: a aplicação inicia, conecta ao banco e executa migrações de maneira reproduzível.
Etapa 2 — Autenticação e isolamento

    Cadastro.
    Login e logout.
    Sessões seguras.
    Perfil do usuário.
    Middleware de autenticação.
    Autorização por proprietário.
    Testes com usuários distintos.

Critério de conclusão: usuários conseguem acessar suas contas sem visualizar ou modificar dados alheios.
Etapa 3 — Contas e movimentações

    Contas financeiras.
    Categorias.
    Receitas e despesas.
    CRUD completo.
    Consultas e filtros.
    Transferências.
    Persistência.
    Testes de integridade.

Critério de conclusão: lançamentos persistem e os saldos e fluxos são calculados corretamente.
Etapa 4 — Gastos mensais

    Navegação por mês.
    Resumo de receitas e despesas.
    Categorias.
    Despesas pendentes e pagas.
    Recorrências.
    Orçamentos.
    Comparativos mensais.

Critério de conclusão: o usuário consegue controlar os gastos domésticos de cada mês, editar os registros e consultar os resultados reais.
Etapa 5 — Patrimônio

    Cadastro de ativos.
    Cadastro de passivos.
    Avaliações.
    Cálculo de patrimônio líquido.
    Histórico patrimonial.
    Prevenção de duplicidade de ativos.

Critério de conclusão: o patrimônio reflete os valores registrados e mantém a integridade das informações.
Etapa 6 — Investimentos

    Cadastro de investimentos.
    Movimentações.
    Aportes e resgates.
    Posições e avaliações.
    Métricas financeiras.
    Integração contábil com contas e patrimônio.

Critério de conclusão: a carteira é editável, persistente e seus indicadores seguem regras financeiras documentadas.
Etapa 7 — Dashboard e relatórios

    Indicadores consolidados.
    Gráficos.
    Relatórios.
    Filtros por período.
    Evolução financeira e patrimonial.
    Validação cruzada entre os indicadores e os registros detalhados.

Critério de conclusão: todos os indicadores usam dados reais e não duplicam valores.
Etapa 8 — Qualidade final

    Revisar segurança.
    Executar testes unitários.
    Executar testes de integração.
    Executar testes de isolamento.
    Validar persistência após reinicialização.
    Corrigir falhas.
    Revisar acessibilidade e responsividade.
    Atualizar a documentação.
    Listar limitações remanescentes.

Critério de conclusão: todas as funcionalidades prioritárias estão integradas e testadas, e eventuais limitações estão explicitamente documentadas.
27. Critérios gerais de aceite

O projeto só deverá ser considerado funcionalmente concluído quando todos os critérios abaixo forem atendidos.
Multiusuário

    Usuários diferentes podem cadastrar e gerenciar seus próprios dados.
    Não existe acesso indevido entre contas.
    A autorização é aplicada no backend.
    Relatórios e agregações respeitam o usuário autenticado.

Persistência

    Os dados são armazenados em SQL.
    Os registros sobrevivem a reinicializações.
    As migrações são reproduzíveis.
    Operações relacionadas são atômicas.
    Erros não deixam dados parcialmente gravados.

Controle mensal

    É possível registrar receitas e despesas.
    É possível editar e consultar os próprios lançamentos.
    Os totais mensais são corretos.
    Categorias e filtros funcionam.
    Orçamentos e recorrências seguem regras explícitas.

Patrimônio

    É possível registrar e editar ativos e passivos.
    O patrimônio líquido é calculado corretamente.
    Os valores possuem data de referência.
    Não existe dupla contabilização.

Investimentos

    É possível cadastrar e editar posições.
    Movimentações são persistidas.
    Métricas não confundem aportes com lucros.
    Valores manuais não são apresentados como cotações externas.
    O investimento não é somado duas vezes ao patrimônio.

Interface

    As telas principais estão funcionais.
    Formulários se comunicam com a API.
    Erros são tratados.
    Dados reais são exibidos.
    A interface é responsiva e utilizável.

Qualidade

    Testes relevantes são executados.
    Os principais riscos de segurança são cobertos.
    Não há funcionalidades essenciais implementadas apenas visualmente.
    A documentação de execução está atualizada.
    As limitações conhecidas são identificadas.

28. Instrução final ao agente de desenvolvimento

Implemente este projeto de forma incremental, mas com visão integral do produto.

Não reduza o escopo a um CRUD simples de despesas. O sistema precisa manter coerência entre contas financeiras, gastos mensais, patrimônio e investimentos.

Não priorize a aparência em detrimento da persistência, segurança e integridade financeira.

Não utilize dados simulados como substitutos de funcionalidades que deveriam operar sobre o banco.

Não permita que a edição de registros cause inconsistência em saldos, históricos, transferências ou posições de investimentos.

Não implemente controles de autorização exclusivamente na interface.

Não declare uma etapa concluída sem verificar seus critérios de aceite e executar os testes correspondentes.

Se uma funcionalidade depender de um serviço externo ainda não configurado, implemente a parte local que puder funcionar de maneira real e documente a dependência restante. Não simule uma integração como se estivesse ativa.

Ao final de cada etapa, apresente um resumo objetivo contendo:

    Funcionalidades implementadas.
    Arquivos e módulos relevantes criados ou alterados.
    Migrações realizadas.
    Testes executados e resultados.
    Decisões técnicas importantes.
    Limitações ou pendências reais.
    Próxima etapa de desenvolvimento.

O resultado esperado é uma aplicação financeira robusta, multiusuário, persistente, segura, testável e preparada para evolução contínua.

A prioridade é entregar um sistema correto e utilizável, com regras de negócio claras e dados confiáveis.
