# Servidor ftp básico

Sabe aquela hora que você tá no lab de informática e precisa passar o ~~counter strike~~ trabalho para os colegas, mas a internet é tão lerda que não vale a pena e também você não quer emprestar seu pendrive por que não quer que as pessoas vejam os segredos obscuros nele? Então, acredito eu que este programinha vai te ajudar muito. 

Ele basicamente cria um servidor de arquivos que pode ser acessado pelo navegador, e por ser feito em go, acredito que performance não seja problema e sobre praticidade é só compilar (`mise exec -- go build -o goftpd ./cmd/goftpd`) e levar o binário pra qualquer lugar.

Por padrão ele expõe a pasta onde ele está, por exemplo, se eu colocar ele na home do pendrive e rodar ele, ele vai estar expondo seu pedrive inteiro.

Pra compilar esse carinha você precisa do compilador de go, o compilador vai gerar um binário (no caso do windows, um arquivo .exe) e este arquivo está pronto para usar. Também tem binário pronto no [GitHub Releases](https://github.com/lewtec/goftpd/releases) (linux/mac/windows, amd64/arm64).

## Release

[GoReleaser](https://goreleaser.com) + [svu](https://github.com/caarlos0/svu). Só archives e checksums (sem Homebrew, Docker ou pacotes). Tags sem prefixo `v` ([`.svu.yml`](.svu.yml)).

```bash
mise release          # next (svu) + goreleaser (precisa GITHUB_TOKEN)
mise release patch    # ou major | minor | next
```

CI: [`.github/workflows/autorelease.yml`](.github/workflows/autorelease.yml). Push/PR roda `mise run ci`. `workflow_dispatch` com patch/minor/major tagueia e publica.