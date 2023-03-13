# sao-node

## Build node
	$ git clone git@github.com:SaoNetwork/sao-node.git
	$ cd sao-node
	$ git checkout v0.1.0
	$ make
Two binaries will be generated:
* saonode - gateway/storage node
* saoclient - cliednt to operate data models.

## Prepare accounts & tokens
	$ ./saonode account create
	Enter account name: node001
	Account: node001
	Address: sao1evfnyhkvgkm676s48y4tkuqj2js4eg23e8h2p4
	Mnemonic:
	feel phone vacant level midnight attract student include common medal walnut van famous matrix hunt lesson evolve silk argue mesh affair grid oppose reunion
This account will be used as storage node account.

	$ ./saoclient account create
	Enter account name:client001
	Account: client001
	Address: sao124uad7f4dvpnfre44yv8dh2ztrvkmcd4xgymrz
	Mnemonic:
	walk mercy worry belt often biology arm angle caution seminar exhibit top raw sentence wasp fringe wage vocal learn wide measure sleep lend link
	$ saod tx bank send alice sao1evfnyhkvgkm676s48y4tkuqj2js4eg23e8h2p4 100000000sao
	$ saod tx bank send alice sao124uad7f4dvpnfre44yv8dh2ztrvkmcd4xgymrz 100000000sao
This account will be used as client account.

## Init & start node
	$ ./saonode --chain-address https://rpc-testnet-node0.sao.network:443 init --creator sao1evfnyhkvgkm676s48y4tkuqj2js4eg23e8h2p4
	...
	563D7EB1856FC26B3720313DAE413F184C3C96FD0DA5869ED6B69BD924830FE6
	$ ./saonode --vv run
	  ____                      _   _          _                               _
	 / ___|    __ _    ___     | \ | |   ___  | |_  __      __   ___    _ __  | | __
	 \___ \   / _` |  / _ \    |  \| |  / _ \ | __| \ \ /\ / /  / _ \  | '__| | |/ /
	  ___) | | (_| | | (_) |   | |\  | |  __/ | |_   \ V  V /  | (_) | | |    |   <
	 |____/   \__,_|  \___/    |_| \_|  \___|  \__|   \_/\_/    \___/  |_|    |_|\_\
	...

## Data model operation
	# Create
	$ ./saoclient model create --content '[{"id": 1, "title": "Note 1"}, {"id": 2, "title": "Note 2"}]' -name my_notes
	...
	
	# Load
	$ ./saoclient model load --keyword my_notes
	...
	
	# Generate Patch
	$ ./saoclient model patch-gen --origin '[{"id": 1, "title": "Note 1"}, {"id": 2, "title": "Note 2"}]' --target '[{"id": 1, "title": "Note 1"}, {"id": 2, "title": "Note 2"}, {"id": 3, "title": "Note 3"}, {"id": 4, "title": "Note 4"}, {"id": 5, "title": "Note 5"}, {"id": 6, "title": "Note 6"}]'
	  Patch      : [{"op":"add","path":"/2","value":{"id":3,"title":"Note 3"}},{"op":"add","path":"/3","value":{"id":4,"title":"Note 4"}},{"op":"add","path":"/4","value":{"id":5,"title":"Note 5"}},{"op":"add","path":"/5","value":{"id":6,"title":"Note 6"}}]
	  Target Cid : bafkreieerchgnsjxcmelllftgqgrm7ftusfkbdylhmhx6kjgnfqm2hdvce
	
	# Update
	$ ./saoclient model update --patch '[{"op":"add","path":"/2","value":{"id":3,"title":"Note 3"}},{"op":"add","path":"/3","value":{"id":4,"title":"Note 4"}},{"op":"add","path":"/4","value":{"id":5,"title":"Note 5"}},{"op":"add","path":"/5","value":{"id":6,"title":"Note 6"}}]' --cid bafkreieerchgnsjxcmelllftgqgrm7ftusfkbdylhmhx6kjgnfqm2hdvce --keyword my_notes
	...
	
