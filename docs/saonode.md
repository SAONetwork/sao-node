# NAME

saonode - Command line for sao network node

# SYNOPSIS

saonode

```
[--chain-address]
[--gateway]
[--help|-h]
[--keyring]
[--repo]
[--version|-v]
[--vv]
```

**Usage**:

```
saonode [GLOBAL OPTIONS] command [COMMAND OPTIONS] [ARGUMENTS...]
```

# GLOBAL OPTIONS
```
--chain-address     sao chain api

--gateway           gateway connection

--help, -h          show help

--keyring           account keyring home directory (default: ~/.sao/)

--repo              repo directory for sao storage node (default: ~/.sao-node)

--version, -v       print the version

--vv                enables very verbose mode, useful for debugging the CLI
```
# COMMANDS

## init

initialize a sao network node

_Options_
```
--creator           node's account on sao chain
--multiaddr         nodes' multiaddr (default: /ip4/127.0.0.1/tcp/5153/)
```
## join

join sao network

_Options_
```
--creator           node's account on sao chain
```
## clean

clean up the local datastore

## update

update node information

_Options_
```
--accept-order      whether this node can accept shard as a storage node
--creator           node's account on sao chain
--multiaddrs        node's multiaddrs
```
## peers

show p2p peer list

## run

start node

## api-token-gen

Generate API tokens

## migrate


## info

show node information

_Options_
```
--creator           node's account on sao chain
```
## claim

claim sao network storage reward

_Options_
```
--creator           node's account on sao chain
```
## job


### orders

orders management

#### status


#### list

List orders

### shards

shards management

#### status

show specified shard status

_Options_
```
--cid               
--orderId            (default: 0)
```
#### list

List shards

### migrations

migration job management

#### list

List migration jobs

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
