# NAME

saoclient - command line for sao network client

# SYNOPSIS

saoclient

```
[--chain-address]
[--gateway]
[--help|-h]
[--keyring]
[--platform]
[--repo]
[--version|-v]
[--vv]
```

**Usage**:

```
saoclient [GLOBAL OPTIONS] command [COMMAND OPTIONS] [ARGUMENTS...]
```

# GLOBAL OPTIONS
```
--chain-address     sao chain api

--gateway           gateway connection

--help, -h          show help

--keyring           account keyring home directory (default: ~/.sao/)

--platform          platform to manage the data model

--repo              repo directory for sao client (default: ~/.sao-cli)

--version, -v       print the version

--vv                enables very verbose mode, useful for debugging the CLI
```
# COMMANDS

## init

initialize a cli sao client

    if you want to use sao cli client, you must first init using this command.
     create sao chain account locally which will be used as default account in following commands. 
    under --repo directory, there are client configuration file and keystore.

_Options_
```
--chain-id           (default: sao)
--key-name, -k      sao chain account key name
```
## net

network management

### info

get peer info of the gateway

### token-gen

generate token to access http file server

### list

list the nodes in SAO Network

## model

data model management

>model related commands including create, update, update permission, etc.

### create

create a new data model

_Options_
```
--cid               data content cid, make sure gateway has this cid file before using this flag. you must either specify --content or --cid. 
--client-publish    true if client sends MsgStore message on chain, or leave it to gateway to send
--content           data model content to create. you must either specify --content or --cid
--delay             how many epochs to wait for the content to be completed storing (default: 60)
--duration          how many days do you want to store the data (default: 365)
--extend-info       extend information for the model
--name              alias name for this data model, this alias name can be used to update, load, etc.
--public            
--replica           how many copies to store (default: 1)
--rule              
--tags              
```
### patch-gen

generate data model patch

>used to before update cmd. you will get patch diff and target cid.

_Options_
```
--origin            the original data model content
--target            the target data model content
```
### update

update an existing data model

>use patch cmd to generate --patch flag and --cid first. permission error will be reported if you don't have model write perm

_Options_
```
--cid               target content cid
--client-publish    true if client sends MsgStore message on chain, or leave it to gateway to send
--commit-id         data model's last commit id
--delay             how many epochs to wait for data update complete (default: 60)
--duration          how many days do you want to store the data. (default: 365)
--extend-info       extend information for the model
--force             overwrite the latest commit
--keyword           data model's alias name, dataId or tag
--patch             patch to apply for the data model
--replica           how many copies to store. (default: 1)
--rule              
--size              target content size (default: 0)
--tags              
```
### update-permission

update data model's permission

>only data model owner can update permission

_Options_
```
--data-id           data model's dataId
--readonly-dids     DIDs with read access to the data model
--readwrite-dids    DIDs with read and write access to the data model
```
### load

load data model

>only owner and dids with r/rw permission can load data model.

_Options_
```
--commit-id         data model's commitId
--dump              dump data model content to ./<dataid>.json
--keyword           data model's alias, dataId or tag
--version           data model's version. you can find out version in commits cmd
```
### delete

delete data model

_Options_
```
--data-id           data model's dataId
```
### commits

list data model historical commits

_Options_
```
--keyword           data model's alias, dataId or tag
```
### list

check models' status

_Options_
```
--date              updated date of data model's to be list
```
### renew

renew data model

_Options_
```
--client-publish    true if client sends MsgStore message on chain, or leave it to gateway to send
--data-ids          data model's dataId list
--delay             how long to wait for the file ready (default: 60)
--duration          how many days do you want to renew the data. (default: 365)
```
### status

check models' status

_Options_
```
--data-ids          data model's dataId list
```
### meta

check models' meta information

_Options_
```
--data-id           data model's dataId
```
### order

check models' order information

_Options_
```
--order-id          data model's orderId (default: 0)
```
## file

file management

### create

ModelCreate a file

_Options_
```
--cid               
--client-publish    true if client sends MsgStore message on chain, or leave it to gateway to send
--delay             how many epochs to wait for the file ready (default: 60)
--duration          how many days do you want to store the data. (default: 365)
--extend-info       extend information for the model
--file-name         local file path
--replica           how many copies to store. (default: 1)
--rule              
--tags              
```
### upload

upload file(s) to storage network

_Options_
```
--filepath          file's path to upload
--multiaddr         remote multiaddr
```
### download

download file(s) from storage network

_Options_
```
--commit-id         file commitId
--keywords          storage network dataId(s) of the file(s)
--version           file version
```
## did

did management

### create

create a new did based on the given sao account.

_Options_
```
--chain-id           (default: sao)
--key-name          sao chain key name which did will be generated on
--override          override default client configuration's key account.
```
### info

show did information

_Options_
```
--did-url           did URL
```
### sign

using the given did to sign a payload

_Options_
```
--key-name          sao chain key name which did will be generated on
```
## account

account management

### list

list all sao chain account in local keystore

### create

create a new local account with the given name

_Options_
```
--key-name          account name
```
### send

send SAO tokens from one account to another

_Options_
```
--amount            the token amount to send (default: 0)
--from              the original account to spend tokens
--to                the target account to received tokens
```
### import


_Options_
```
--key-name          account name to import
```
### export

Export the given local account's encrypted private key

_Options_
```
--key-name          account name to export
```
## clidoc


_Options_
```
--doctype           current supported type: markdown / man (default: markdown)
--help, -h          show help
--output            file path to export to
```
### help, h

Shows a list of commands or help for one command

## help, h

Shows a list of commands or help for one command
