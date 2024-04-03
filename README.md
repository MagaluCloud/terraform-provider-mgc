## Magalu Cloud Provider

O provider MGC permite que você utilize Terraform ou OpenTofu para gerenciar seus recursos na Magalu Cloud.

Com o provider você pode gerenciar:

- VPCs (sub-redes, regras de segurança, IPs públicos)
- Virtual Machines (instâncias, snapshots)
- Kubernetes (clusters, nodepools)
- DBaaS (instâncias, replicações)
- Block Storage (volumes, snapshots, anexação em VM)

O provider está em fase de desenvolvimento, então novos recursos da Magalu Cloud serão suportados em breve.


## Pré-Requisitos

Antes de começar tenha certeza de que você tem o Terraform instalado e para isso siga as [instruções no site da Hashicorp](https://developer.hashicorp.com/terraform/install).

## Download e Instalação

Baixe a *release* correta para seu sistema e arquitetura no link abaixo.

[Releases](https://github.com/MagaluCloud/terraform/releases/)

Atualmente o provider MGC Terraform não está disponível no Terraform registry (online). Os passos abaixo auxiliam na instalação do provider localmente usando o método dev_override.

2. Descompacte o arquivo para o local de sua preferência. 

3. Crie uma pasta para o seu projeto, como por exemplo: **~/terraform-mgc**.

4. Copie para esta pasta o binário do provider como por exemplo: **terraform-provider-mgc**. 

5. Crie um novo arquivo de configuração para o projeto, como por exemplo **~/terraform-mgc/main.tf** e adicione o seguinte conteúdo ao arquivo:

```
// main.tf
   terraform {
    required_providers {
        mgc = {
              source = "registry.terraform.io/magalucloud/mgc"
              version = "0.18.0"
        }
    }
   }
   provider "mgc" {}  
```

Lembre de atualizar o número da versão para a versão atual da sua CLI.

### Ajustando link para o provider 
  
Como estamos usando a versão offline do provider, precisamos ajustar a sua referência para a pasta onde ele está instalado. Para isso adicione o seguinte conteúdo no arquivo de configuração local de seu usuário: **~/.terraformrc**. 

No Windows esse arquivo está localizado em **$env:APPDATA** e seu nome deve ser terraform.rc . 

```java
// terraform.rc
provider_installation {
  dev_overrides {
    "registry.terraform.io/magalucloud/mgc" = "/home/usuario/terraform-mgc"
  }

  direct {}
}  
```

Caso o arquivo de configuração não exista, você deverá criá-lo previamente. 
  
Nos sistemas Windows o caminho para a pasta terraform-mgc deve ser referenciado desta forma:  
  
C:/Users/<seu_usuario>/terraform-mgc 
  
Para mais informações sobre o arquivo de configuração, veja a documentação oficial da Hashicorp aqui.

### Autenticação

É importante que você tenha feito o procedimento de autenticação pelo CLI ao menos uma vez, porque o provider Terraform utilizará a mesmo token de sessão do usuário armazenados localmente.

Usando a CLI você deve executar o seguinte comando em um terminal:

```
mgc auth login
```

### Adicionando seus recursos 

Para adicionar recursos suportados da Magalu Cloud no seu projeto terraform, siga as instruções contidas na referência que está localizada dentro do mesmo arquivo ZIP onde estava o binário, na pasta **doc/resources**.

Abaixo segue um exemplo de configuração para instância de Virtual Machine: 

```java
// virtual_machines.tf
resource "mgc_virtual-machine_instances" "myvm" {
   provider = "mgc"
   name = "my-tf-vm"

   machine_type = {
       name = "cloud-bs1.xsmall"
   }

   image = {
       name = "cloud-ubuntu-22.04 LTS"
   }

   ssh_key_name = "my_ssh_key"

   availability_zone = "br-ne-1c"
}  
```

Esse exemplo deve ser adicionado a um arquivo com extensão .tf , como por exemplo **virtual_machines.tf** e deve ficar na mesma pasta do arquivo **main.tf** criado anteriormente. 

### Testando seu projeto terraform 

Agora vamos executar o seu projeto terraform. Os comandos sugeridos abaixo seguem o fluxo básico de execução e verificação da ferramenta, mas para melhor compreender o seu funcionamento recomendamos visitar a documentação oficial a página do desenvolvedor: [Terraform Documentation](https://developer.hashicorp.com/terraform/docs) 

Como estamos utilizando um provider local, **não é necessário** rodar o comando _terraform init_. 

Abra um seu terminal dentro da pasta onde está o projeto terraform (os arquivos **.tf**) e execute o seguinte comando para revisar as mudanças antes de aplicar: 

```
terraform plan
```

E quando estiver seguro das alterações execute o comando abaixo para aplicá-las.

```
terraform apply
```

Responda aos prompts de confirmação, caso sejam exibidos no seu terminal. 

Para verificar as alterações aplicadas sobre seus recursos na Magalu Cloud, execute o comando abaixo. 

```
terraform show
```