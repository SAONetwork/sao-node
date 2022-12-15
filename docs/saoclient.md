# NAME

saoclient - command line for sao network client

# SYNOPSIS

saoclient

```
[--chain-address]
[--gateway]
[--help|-h]
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

**Options**
```
--key-name, -k      sao chain account key name
```
## model

data model management

>model related commands including create, update, update permission, etc.

### create

create a new data model

**Options**
```
--cid               data content cid, make sure gateway has this cid file before using this flag. you must either specify --content or --cid. 
--client-publish    true if client sends MsgStore message on chain, or leave it to gateway to send
--content           data model content to create. you must either specify --content or --cid
--delay             how many epochs to wait for the content to be completed storing (default: 86400)
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

**Options**
```
--origin            the original data model content
--target            the target data model content
```
### update

update an existing data model

>use patch cmd to generate --patch flag and --cid first. permission error will be reported if you don't have model write perm

**Options**
```
--cid               target content cid
--client-publish    true if client sends MsgStore message on chain, or leave it to gateway to send
--delay             how many epochs to wait for data update complete (default: 86400)
--duration          how many days do you want to store the data. (default: 365)
--extend-info       extend information for the model
--force             overwrite the latest commit
--keyword           data model's alias name, dataId or tag
--patch             patch to apply for the data model
--replica           how many copies to store. (default: 1)
--rule              
--tags              
```
### update-permission

update data model's permission

>only data model owner can update permission

**Options**
```
--data-id           data model's dataId
--readonly-dids     DIDs with read access to the data model
--readwrite-dids    DIDs with read and write access to the data model
```
### load

load data model

>only owner and dids with r/rw permission can load data model.

**Options**
```
--commit-id         data model's commitId
--dump              dump data model content to ./<dataid>.json
--keyword           data model's alias, dataId or tag
--version           data model's version. you can find out version in commits cmd
```
### delete

delete data model

**Options**
```
--data-id           data model's dataId
```
### commits

list data model historical commits

**Options**
```
--keyword           data model's alias, dataId or tag
```
### renew

renew data model

**Options**
```
--client-publish    true if client sends MsgStore message on chain, or leave it to gateway to send
--data-ids          data model's dataId list
--delay             how long to wait for the file ready (default: 86400)
--duration          how many days do you want to renew the data. (default: 365)
```
### status

check models' status

**Options**
```
--data-ids          data model's dataId list
```
## file

file management

### create

Create a file

**Options**
```
--cid               
--client-publish    true if client sends MsgStore message on chain, or leave it to gateway to send
--delay             how many epochs to wait for the file ready (default: 86400)
--duration          how many days do you want to store the data. (default: 365)
--extend-info       extend information for the model
--file-name         local file path
--replica           how many copies to store. (default: 1)
--rule              
--tags              
```
### peer-info

get peer info of the gateway

### token-gen

generate token to access http file server

### upload

upload file(s) to storage network

**Options**
```
--filepath          file's path to upload
--multiaddr         remote multiaddr
```
### download

download file(s) from storage network

**Options**
```
--commit-id         file commitId
--keywords          storage network dataId(s) of the file(s)
--version           file version
```
## did

did management

### create

create a new did based on the given sao account.

**Options**
```
--key-name          sao chain key name which did will be generated on
--override          override default client configuration's key account.
```
### sign

using the given did to sign a payload

**Options**
```
--key-name          sao chain key name which did will be generated on
```
## account

account management

### list

list all sao chain account in local keystore

### create

create a new local account with the given name

**Options**
```
--key-name          account name
```
### import


**Options**
```
--key-name          account name to import
```
### export

Export the given local account's encrypted private key

**Options**
```
--key-name          account name to export
```
## help, h

Shows a list of commands or help for one command
