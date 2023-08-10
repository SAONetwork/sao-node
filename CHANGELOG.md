<a name="unreleased"></a>
## [Unreleased]


<a name="v0.1.7"></a>
## [v0.1.7](https://github.com/SAONetwork/sao-node.git/compare/v0.1.6...v0.1.7) (2023-08-08)

### Features

* implement model list ([#38](https://github.com/SAONetwork/sao-node.git/issues/38))  *#38* 
* create model with public access ([#32](https://github.com/SAONetwork/sao-node.git/issues/32))  *#32* 
* add tcp rpc handler ([#35](https://github.com/SAONetwork/sao-node.git/issues/35))  *#35* 
* add http lru cache ([#33](https://github.com/SAONetwork/sao-node.git/issues/33))  *#33* 

### Bug Fixes

* libp2p tcp client support ([#41](https://github.com/SAONetwork/sao-node.git/issues/41))  *#2*  *#41*  *#2* 
* load content type ([#39](https://github.com/SAONetwork/sao-node.git/issues/39))  *#39*  *#2* 
* gateway disable storage module bug fix - http server not working ([#34](https://github.com/SAONetwork/sao-node.git/issues/34))  *#34* 


<a name="v0.1.6"></a>
## [v0.1.6](https://github.com/SAONetwork/sao-node.git/compare/v0.1.5...v0.1.6) (2023-07-12)

### Features

* add version tag ([#27](https://github.com/SAONetwork/sao-node.git/issues/27))  *#27* 
* add query faults cmd in saonode side ([#26](https://github.com/SAONetwork/sao-node.git/issues/26))  *#26* 
* load file via Http ([#28](https://github.com/SAONetwork/sao-node.git/issues/28))  *#28* 

### Bug Fixes

* try fetch content from all shards, filter out timeout shards in timeout-retry tasks ([#25](https://github.com/SAONetwork/sao-node.git/issues/25))  *#25* 
* add default token during client init 
* validate accounce addresses 

### Code Refactoring

* update module name to github.com/SaoNetwork/sao-node ([#29](https://github.com/SAONetwork/sao-node.git/issues/29))  *#29* 


<a name="v0.1.5"></a>
## [v0.1.5](https://github.com/SAONetwork/sao-node.git/compare/v0.1.4...v0.1.5) (2023-06-29)

### Features

* update shard job to expire status after expiration ([#23](https://github.com/SAONetwork/sao-node.git/issues/23))  *#23* 
* add storage command ([#21](https://github.com/SAONetwork/sao-node.git/issues/21))  *#21* 
* add rpc to load model delegated by gateway ([#20](https://github.com/SAONetwork/sao-node.git/issues/20))  *#20* 
* poe ([#17](https://github.com/SAONetwork/sao-node.git/issues/17))  *#17* 
* remove expire shard from storage ([#19](https://github.com/SAONetwork/sao-node.git/issues/19))  *#19* 

### Bug Fixes

* meta status ([#22](https://github.com/SAONetwork/sao-node.git/issues/22))  *#22* 
* add startup pledge check ([#18](https://github.com/SAONetwork/sao-node.git/issues/18))  *#18* 
* fetch replica mode content, handle invalid cid, model cache evict ([#13](https://github.com/SAONetwork/sao-node.git/issues/13))  *#13* 


<a name="v0.1.4"></a>
## [v0.1.4](https://github.com/SAONetwork/sao-node.git/compare/v0.1.3...v0.1.4) (2023-04-21)

### Features

* optimize node api connection ([#9](https://github.com/SAONetwork/sao-node.git/issues/9))  *#9* 
* handle timeout order 

### Bug Fixes

* node info print, total count in ListMeta/ListShards, check metadata commits length in QueryMeta 
* use default path, remove StagingPath, IpfsRepo and httpServerPath from config 
* avoid tx duplicate nonce from same account 
* move did print into did info cmd 

### Code Refactoring

* support indexing and graphql ([#7](https://github.com/SAONetwork/sao-node.git/issues/7))  *#7* 


<a name="v0.1.3"></a>
## [v0.1.3](https://github.com/SAONetwork/sao-node.git/compare/v0.1.2...v0.1.3) (2023-03-15)

