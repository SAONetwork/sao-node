<a name="unreleased"></a>
## [Unreleased]

### Features
- update shard job to expire status after expiration
- add storage command ([#21](https://github.com/SAONetwork/sao-node.git/issues/21))
- add rpc to load model delegated by gateway ([#20](https://github.com/SAONetwork/sao-node.git/issues/20))
- poe ([#17](https://github.com/SAONetwork/sao-node.git/issues/17))
- remove expire shard from storage ([#19](https://github.com/SAONetwork/sao-node.git/issues/19))

### Bug Fixes
- meta status ([#22](https://github.com/SAONetwork/sao-node.git/issues/22))
- add startup pledge check ([#18](https://github.com/SAONetwork/sao-node.git/issues/18))
- fetch replica mode content, handle invalid cid, model cache evict ([#13](https://github.com/SAONetwork/sao-node.git/issues/13))


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

