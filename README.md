# pxma
Zero trust with OpenZiti experiments

## Preperation

### SSH

Create ssh config in `~/.ssh/config`:

```
Host pxma
   HostName pxma-01.test.rz.htwg-konstanz.de
   IdentityFile ~/.ssh/pxma/id_ed25519
   User root

Host pxma-*
   IdentityFile ~/.ssh/pxma/id_ed25519
   User root

Host pxma-01
   HostName pxma-01.test.rz.htwg-konstanz.de

Host pxma-02
   HostName pxma-02.test.rz.htwg-konstanz.de

Host pxma-03
   HostName pxma-03.test.rz.htwg-konstanz.de

Host pxma-04
   HostName pxma-04.test.rz.htwg-konstanz.de

Host pxma-05
   HostName pxma-05.test.rz.htwg-konstanz.de

Host vm01.pxma
   ProxyJump pxma
   HostName 10.0.0.10
   User debian
```

### Packages

To use this package you need to install Ansible and hcloud cli programm. Make sure you have the required python and ansible roles and packages installed.

```bash
> pip install -r requirements.txt
> ansible-galaxy install -r requirements.yml
```

### Setup

Go into this repository root and create a directory called `.private`. This
directory is not ignored by git and should protect automatically commiting your
secrects. Create a new file called `vars` inside the `.private` directory and
copy the following content and fill in your data:

```bash
#!/bin/bash

export HCLOUD_TOKEN="hcloud_token"
export HCLOUD_SSH_KEY="ssh_key"
export PXMA_EMAIL="your_email_for_letsencrypt"
export ZITI_ADMIN_PW="admin_passwd"

```

Before you do any actions, execute the following command to populate these
environment variables in your current terminal session. This simplifies testing
very much.

```bash
> source .private/vars
```

## Install

### 1. Step: Install the controller

The controller installation requires the creation of a new virtual machine on
Hetzner. Then, the controller ansible script can run to provision the
controller.

```bash
> hcloud server create --label "controller=" --name "pxma01" --type "cx22" --location "nbg1" --image ubuntu-24.04 --ssh-key $HCLOUD_SSH_KEY
```

Create DNS entries with the IP-address of for the new server. 

```bash
> ansible-playbook -i inventory/testing controller.yaml
```
