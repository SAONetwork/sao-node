# NAME

saonode - Command line for sao network node

# SYNOPSIS

saonode

```
[--chain-address]
[--help|-h]
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

--help, -h          show help

--repo              repo directory for sao storage node (default: ~/.sao-node)

--version, -v       print the version

--vv                enables very verbose mode, useful for debugging the CLI
```
# COMMANDS

## init

initialize a sao network node

**Options**
```
--creator           node's account on sao chain
--multiaddr         nodes' multiaddr (default: /ip4/127.0.0.1/tcp/5153/)
```
## join

join sao network

**Options**
```
--creator           node's account on sao chain
```
## reset

update node information

**Options**
```
--accept-order      whether this node can accept shard as a storage node
--creator           node's account on sao chain
--multiaddrs        node's multiaddrs
```
## peers

show p2p peer list

## quit

quit sao network

>can re-join sao network by 'join' cmd. after quiting, no new shard will be assign to this node.

**Options**
```
--creator           node's account on chain
```
## run

start node

## api-token-gen

Generate API tokens

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
