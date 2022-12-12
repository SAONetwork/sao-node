# Node

SAO Network is consist of gateway nodes and storage nodes. Gateway node and storage node can be combined into one node.

* 

## Node Management

* .sao-node/
   - http-files
   - ipfs
   - staging
* .sao-cli



#### init

initialize a node and join sao network.

```bash
$ saonode init --creator <cosmos address> --repo <repo>
<tx hash>
```

this command will:

1. generate p2p information
2. register into sao network on chain to be eligible to serve future store/load tasks.
3. node repo initialization.

node repo will be auto-generated as below structure:

```bash
├── config.toml
├── datastore
└── keystore
    └── libp2p.key
```

* config.toml: default configuration file
* datastore: store node data in database
* keystore/libp2p.key: p2p connection key.

#### quit

quit sao network. no tasks will be dispatched to quited node any more.

```bash
$ saonode quit --creator <cosmos address>
```

#### join

rejoin sao network to be eligible to serve task again.

````bash
$ saonode join --creator <cosmos address>
````

#### reset

update node information on chain.

```bash
$ saonode reset --creator <cosmos address> --multiaddrs <ma>
```

#### peer

list node connected p2p peers

```bash
$ saonode peers
```

#### api-token-gen



#### run

start node

```bash
$ saonode run
  ____                      _   _          _                               _
 / ___|    __ _    ___     | \ | |   ___  | |_  __      __   ___    _ __  | | __
 \___ \   / _` |  / _ \    |  \| |  / _ \ | __| \ \ /\ / /  / _ \  | '__| | |/ /
  ___) | | (_| | | (_) |   | |\  | |  __/ | |_   \ V  V /  | (_) | | |    |   <
 |____/   \__,_|  \___/    |_| \_|  \___|  \__|   \_/\_/    \___/  |_|    |_|\_\
...
2022-12-09T18:10:52.820+0800	INFO	node	node/node.go:214	storage node initialized
...
2022-12-09T18:10:52.821+0800	INFO	node	node/node.go:260	gateway node initialized
```

## Gateway Node



1. prepare cosmos account with some balance.

2. initialize

```bash
$ saonode init --creator <cosmos address>
```

3. configure it to be a gateway node

   ```yaml
   [Module]
   StorageEnable = false
   
   [SaoIpfs]
   Enable = false
   ```

if your gateway is only used to serve load request, disable accept-order in config.toml.

```yaml
[Storage]
AcceptOrder = false
```

4. start gateway

```bash
$ saonode run
  ____                      _   _          _                               _
 / ___|    __ _    ___     | \ | |   ___  | |_  __      __   ___    _ __  | | __
 \___ \   / _` |  / _ \    |  \| |  / _ \ | __| \ \ /\ / /  / _ \  | '__| | |/ /
  ___) | | (_| | | (_) |   | |\  | |  __/ | |_   \ V  V /  | (_) | | |    |   <
 |____/   \__,_|  \___/    |_| \_|  \___|  \__|   \_/\_/    \___/  |_|    |_|\_\
...
2022-12-09T18:18:31.383+0800	INFO	node	node/node.go:260	gateway node initialized
```

 if you see "gateway node initialized", your gateway start successfully.



## Storage Node

1. prepare cosmos account with enough balance.
2. initialize

```
$ saonode init
```

3. configure it to be a storage node by disabling gateway feature.

```
[Module]
GatewayEnable = false
Ipfs = []

[SaoIpfs]
Enable = true
Repo = "~/.sao-ipfs"
```

if you want your node to have ipfs in process, set Enable to true in SaoIpfs section and set a IPFS repo path.

If you have remote ipfs storages, set them in Ipfs array in Module section.

4. start storage

```
$ saonode run
  ____                      _   _          _                               _
 / ___|    __ _    ___     | \ | |   ___  | |_  __      __   ___    _ __  | | __
 \___ \   / _` |  / _ \    |  \| |  / _ \ | __| \ \ /\ / /  / _ \  | '__| | |/ /
  ___) | | (_| | | (_) |   | |\  | |  __/ | |_   \ V  V /  | (_) | | |    |   <
 |____/   \__,_|  \___/    |_| \_|  \___|  \__|   \_/\_/    \___/  |_|    |_|\_\
...
2022-12-09T18:26:56.000+0800	INFO	node	node/node.go:214	storage node initialized
```

 if you see "storage node initialized", your storage node start successfully.



# Client

#### preparation

1. a local cosmos account:

```bash
$ saoclient account create
Enter account name:testuser
ChainId:  sao-testnet-fcf77b
Account: testuser
Address: cosmos1fedjcvhrk4agdf63rtzxzsk68jqddnkre4xdd6
Mnemonic:
stool rug blame artwork stereo resource artefact gallery permit mail carry pitch truck thing giraffe you prepare kitten february stairs oxygen aunt skirt tray
```

2. a did as data owner

```bash
$ saoclient --net devnet did create --key-name testuser
2022-12-09T18:43:00.844+0800	INFO	chain	chain/chain.go:56	initialize chain client
2022-12-09T18:43:00.846+0800	INFO	chain	chain/chain.go:70	initialize chain listener
Created DID did:key:zQ3shXWxQcZWLYENfBrg4B3iv8kdUxsEQYCbJg3x6dCr1rC4P. tx hash 9491126617A8C19A2BD548F214BE7883CF1C5B4474A2B756C558600FB0EA12BC
```

the generated did did:key:zQ3shXWxQcZWLYENfBrg4B3iv8kdUxsEQYCbJg3x6dCr1rC4P can be used as data owner.

#### create model

```bash
$ saoclient model create --content '[{"id": 1, "title": "Note 1"}, {"id": 2, "title": "Note 2"}]' -name my_notes
alias: my_notes, data id: 612a9104-ed58-4ebe-9c90-d1f3d609f2fa
```

#### patch model

```bash
$ saoclient model patch-gen --origin '[{"id": 1, "title": "Note 1"}, {"id": 2, "title": "Note 2"}]' --target '[{"id": 3, "title":"Note 3"}]'
  Patch      : [{"op":"remove","path":"/1"},{"op":"remove","path":"/0"},{"op":"add","path":"/0","value":{"id":3,"title":"Note 3"}}]
  Target Cid : bafkreifzcud7fnbzia5rtci253jntjqwtxn532rmo3fqpvtszaygoxhxke
```

#### update model

```bash
$ saoclient model update --keyword my_notes --patch '[{"op":"remove","path":"/0/title"},{"op":"replace","path":"/0/id","value":4}]'  --cid bafkreih2glvg5xml2tpzs5ivewmtt7wpdl6gvqep6ozmvtt62qdfspz4ti
alias: my_notes, data id: 612a9104-ed58-4ebe-9c90-d1f3d609f2fa, commitId: 40b425b0-46a6-4fa1-a1e3-1859900421a9.
```

#### commits

```bash
$ saoclient model commits --keyword my_notes
  Model DataId : 612a9104-ed58-4ebe-9c90-d1f3d609f2fa
  Model Alias  : my_notes
  -----------------------------------------------------------
  Version |Commit                              |Height
  -----------------------------------------------------------
  v0	  |612a9104-ed58-4ebe-9c90-d1f3d609f2fa|4082
  v1	  |6e709d08-1133-449c-a111-ad99afdc526d|4391
  v2	  |40b425b0-46a6-4fa1-a1e3-1859900421a9|4652
  -----------------------------------------------------------
```

#### renew model

```bash
$ saoclient model renew --data-ids 612a9104-ed58-4ebe-9c90-d1f3d609f2fa
successfully renewed model[612a9104-ed58-4ebe-9c90-d1f3d609f2fa] with orderId[9].
```

#### load model

````bash
$ saoclient model load --keyword my_notes                                                                       
2022-12-12T12:29:47.558+0800	INFO	chain	chain/chain.go:56	initialize chain client
2022-12-12T12:29:47.573+0800	INFO	chain	chain/chain.go:70	initialize chain listener
  DataId    : 612a9104-ed58-4ebe-9c90-d1f3d609f2fa
  Alias     : my_notes
  CommitId  : 612a9104-ed58-4ebe-9c90-d1f3d609f2fa
  Version   : v0
  Cid       : bafkreidw636luhqz7f3u2hj5e2w2xij5lplxdrycuznntxrimuojiillly
  Content   : [{"id": 1, "title": "Note 1"}, {"id": 2, "title": "Note 2"}]
➜  sao-storage-node git:(data_model_dev) ./saoclient model load --keyword 612a9104-ed58-4ebe-9c90-d1f3d609f2fa        
2022-12-12T12:30:04.006+0800	INFO	chain	chain/chain.go:56	initialize chain client
2022-12-12T12:30:04.017+0800	INFO	chain	chain/chain.go:70	initialize chain listener
  DataId    : 612a9104-ed58-4ebe-9c90-d1f3d609f2fa
  Alias     : my_notes
  CommitId  : 612a9104-ed58-4ebe-9c90-d1f3d609f2fa
  Version   : v0
  Cid       : bafkreidw636luhqz7f3u2hj5e2w2xij5lplxdrycuznntxrimuojiillly
  Content   : [{"id": 1, "title": "Note 1"}, {"id": 2, "title": "Note 2"}]
````



#### update permission

```bash
$ saoclient model update-permission --readonly-dids did:key:zQ3shYVX2m4CJKRAm2vbTUnjRWcnwW22ZHRhS6GhHYk7xUdjz --readwrite-dids did:key:zQ3shsfykXBJGhueLVH3f7ts23vRs6C2wzUAfYaQfQchXiceF --data-id 612a9104-ed58-4ebe-9c90-d1f3d609f2fa
```



# port
* gateway: 5151
* http: 5152
* p2p: tcp 5153/udp 5154